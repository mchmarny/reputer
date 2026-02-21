package github

import (
	"fmt"
	"net/http"
	"os"
	"time"

	hub "github.com/google/go-github/v52/github"
	"github.com/gregjones/httpcache"
	"golang.org/x/oauth2"
)

const httpTimeout = 30 * time.Second

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
