package github

import (
	"net/http"
	"os"

	api "github.com/google/go-github/v52/github"
	"github.com/gregjones/httpcache"
	"golang.org/x/oauth2"
)

const (
	tokenEnvVar = "GITHUB_TOKEN"
)

// getClient returns a GitHub client.
func getClient() *api.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: os.Getenv(tokenEnvVar),
		},
	)
	tc := &http.Client{
		Transport: &oauth2.Transport{
			Base:   httpcache.NewMemoryCacheTransport(),
			Source: ts,
		},
	}

	return api.NewClient(tc)
}
