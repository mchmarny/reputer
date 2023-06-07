package scanner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"github.com/mchmarny/reputer/pkg/pager"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	maxIdleConns     = 10
	timeoutInSeconds = 60
	maxPageSize      = 1000

	clientID = "reputer"
	queryURL = "https://api.osv.dev/v1/query"
	batchURL = "https://api.osv.dev/v1/querybatch"
	vulnURL  = "https://api.osv.dev/v1/vulns/%s"
)

var (
	reqTransport = &http.Transport{
		MaxIdleConns:          maxIdleConns,
		IdleConnTimeout:       timeoutInSeconds * time.Second,
		DisableCompression:    true,
		DisableKeepAlives:     false,
		ResponseHeaderTimeout: time.Duration(timeoutInSeconds) * time.Second,
	}
)

func getClient() *http.Client {
	return &http.Client{
		Timeout:   time.Duration(timeoutInSeconds) * time.Second,
		Transport: reqTransport,
	}
}

// BatchQuery returns a list of vulnerabilities for a list of commit.
func BatchQuery(ctx context.Context, list []*Request) (map[string]int, error) {
	if list == nil {
		return nil, errors.New("req is nil")
	}

	p, err := pager.GetPager(list, maxPageSize)
	if err != nil {
		return nil, errors.Wrap(err, "error creating pager")
	}

	log.Debugf("batch querying %d items...", len(list))
	ids := make([]string, 0)

	for {
		items := p.Next()
		if len(items) < 1 {
			break
		}

		req2 := &BatchRequest{
			Queries: items,
		}

		r, err := query[BatchResult](ctx, batchURL, req2)
		if err != nil {
			return nil, errors.Wrap(err, "error querying")
		}

		log.Debugf("page: %d, results: %d", p.GetCurrentPage(), len(r.Results))

		for _, rez := range r.Results {
			for _, v := range rez.Vulnerabilities {
				ids = append(ids, v.ID)
			}
		}
	}

	log.Debugf("ids: %d", len(ids))
	var wg sync.WaitGroup
	vulns := make(map[string]int)

	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			v, err := getVuln(ctx, fmt.Sprintf(vulnURL, id))
			if err != nil {
				log.Errorf("error getting vulnerability %s: %v", id, err)
				return
			}
			for _, c := range v.Affected {
				for dk, dv := range c.Data {
					log.Debugf("data (key: %s, value: %s)", dk, dv)
					if dk == "severity" {
						vulns[dv.(string)]++
					}
				}
			}
		}(id)
	}

	return vulns, nil
}

// Query returns a list of vulnerabilities for a given commit.
func Query(ctx context.Context, commit string) (*RequestResult, error) {
	if commit == "" {
		return nil, errors.New("commit is empty")
	}

	q := &Request{
		Commit: commit,
	}

	r, err := query[RequestResult](ctx, queryURL, q)
	if err != nil {
		return nil, errors.Wrap(err, "error querying")
	}

	return r, nil
}

func query[T any](ctx context.Context, url string, q interface{}) (*T, error) {
	if url == "" {
		return nil, errors.New("url is required")
	}

	b, err := json.Marshal(q)
	if err != nil {
		return nil, errors.Wrap(err, "error marshalling data")
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		return nil, errors.Wrapf(err, "error creating request: %s", url)
	}
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", clientID)

	res, err := getClient().Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "error handling response: %s", url)
	}
	defer res.Body.Close()

	if log.IsLevelEnabled(log.DebugLevel) {
		dump, err := httputil.DumpResponse(res, true)
		if err != nil {
			log.Errorf("error dumping response: %v", err)
		}
		log.Debugf("response: %s", dump)
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("error querying: %s, status: (%s)", url, res.Status)
	}

	var r T
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, errors.Wrapf(err, "error decoding API response from %s", url)
	}

	return &r, nil
}

func getVuln(ctx context.Context, id string) (*Vulnerability, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}

	url := fmt.Sprintf(vulnURL, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating request: %s", url)
	}

	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", clientID)
	res, err := getClient().Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "error handling response: %s", url)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		if log.IsLevelEnabled(log.DebugLevel) {
			dump, err := httputil.DumpResponse(res, true)
			if err != nil {
				log.Errorf("error dumping response: %v", err)
			}
			log.Debugf("response: %s", dump)
		}
		return nil, errors.Errorf("error querying: %s, status: (%s)", url, res.Status)
	}

	var v Vulnerability
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		return nil, errors.Wrapf(err, "error decoding get API response from %s", url)
	}

	return &v, nil
}
