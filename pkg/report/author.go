// Package report defines the data model for reputation reports.
package report

import (
	"fmt"
)

// MakeAuthor creates a new Author instance.
func MakeAuthor(username string) *Author {
	return &Author{
		Username: username,
		Context:  &AuthorContext{},
		Stats:    &Stats{},
	}
}

// Author represents a commit author.
type Author struct {
	Username   string         `json:"username"`
	Reputation float64        `json:"reputation"`
	Context    *AuthorContext `json:"context,omitempty"`
	Stats      *Stats         `json:"stats,omitempty"`
}

func (a *Author) String() string {
	if a == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", *a)
}

// AuthorContext holds optional author metadata.
type AuthorContext struct {
	Created string `json:"created,omitempty"`
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Company string `json:"company,omitempty"`
}

// Stats represents a set of statistics for an author.
type Stats struct {
	Suspended         bool  `json:"suspended,omitempty"`
	CommitsVerified   bool  `json:"verified_commits,omitempty"`
	StrongAuth        bool  `json:"strong_auth,omitempty"`
	AgeDays           int64 `json:"age_days"`
	Commits           int64 `json:"commits"`
	UnverifiedCommits int64 `json:"unverified_commits"`
	PublicRepos       int64 `json:"public_repos,omitempty"`
	PrivateRepos      int64 `json:"private_repos,omitempty"`
	Followers         int64 `json:"followers,omitempty"`
	Following         int64 `json:"following,omitempty"`
	LastCommitDays    int64 `json:"last_commit_days,omitempty"`
	OrgMember         bool  `json:"org_member,omitempty"`
}
