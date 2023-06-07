package github

import (
	"net/http"
	"os"

	hub "github.com/google/go-github/v52/github"
	"github.com/gregjones/httpcache"
	"golang.org/x/oauth2"
)

// getClient returns a GitHub client.
func getClient() *hub.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: os.Getenv("GITHUB_TOKEN"),
		},
	)
	tc := &http.Client{
		Transport: &oauth2.Transport{
			Base:   httpcache.NewMemoryCacheTransport(),
			Source: ts,
		},
	}

	return hub.NewClient(tc)
}
