package github

import (
	"fmt"
	"net/http"
	"os"
	"time"

	hub "github.com/google/go-github/v72/github"
	"github.com/gregjones/httpcache"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	httpTimeout        = 30 * time.Second
	rateLimitThreshold = 10
)

// getClient returns a GitHub client.
func getClient() (*hub.Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable must be set")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: token,
		},
	)
	tc := &http.Client{
		Timeout: httpTimeout,
		Transport: &oauth2.Transport{
			Base:   httpcache.NewMemoryCacheTransport(),
			Source: ts,
		},
	}

	return hub.NewClient(tc), nil
}

// waitForRateLimit pauses execution when the remaining rate limit is low.
func waitForRateLimit(r *hub.Response) {
	if r == nil {
		return
	}
	if r.Rate.Remaining >= rateLimitThreshold {
		return
	}
	resetAt := r.Rate.Reset.Time
	wait := time.Until(resetAt) + time.Second
	if wait <= 0 {
		return
	}
	log.WithFields(log.Fields{
		"remaining": r.Rate.Remaining,
		"reset_at":  resetAt.Format(time.RFC3339),
		"wait_secs": int(wait.Seconds()),
	}).Warn("rate limit low, waiting for reset")
	time.Sleep(wait)
}
