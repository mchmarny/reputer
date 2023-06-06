package report

import (
	"fmt"
	"time"
)

// MakeAuthor creates a new Author instance.
func MakeAuthor(username string) *Author {
	return &Author{
		Username: username,
		Context:  make(map[string]interface{}),
	}
}

// Author represents a commit author.
type Author struct {
	Username          string                 `json:"username"`
	Created           time.Time              `json:"created"`
	Suspended         bool                   `json:"suspended,omitempty"`
	PublicRepos       int64                  `json:"public_repos,omitempty"`
	PrivateRepos      int64                  `json:"private_repos,omitempty"`
	Followers         int64                  `json:"followers,omitempty"`
	Following         int64                  `json:"following,omitempty"`
	Commits           int64                  `json:"commits"`
	UnverifiedCommits int64                  `json:"-"`
	CommitsVerified   bool                   `json:"verified_commits"`
	StrongAuth        bool                   `json:"strong_auth"`
	Reputation        float64                `json:"reputation"`
	Context           map[string]interface{} `json:"context,omitempty"`
}

func (a *Author) String() string {
	if a == nil {
		return ""
	}
	return fmt.Sprintf("%+v", *a)
}
