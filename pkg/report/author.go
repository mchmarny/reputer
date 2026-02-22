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
	Username   string         `json:"username" yaml:"username"`
	Reputation float64        `json:"reputation" yaml:"reputation"`
	Context    *AuthorContext `json:"context,omitempty" yaml:"context,omitempty"`
	Stats      *Stats         `json:"stats,omitempty" yaml:"stats,omitempty"`
}

func (a *Author) String() string {
	if a == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", *a)
}

// AuthorContext holds optional author metadata.
type AuthorContext struct {
	Created string `json:"created,omitempty" yaml:"created,omitempty"`
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Email   string `json:"email,omitempty" yaml:"email,omitempty"`
	Company string `json:"company,omitempty" yaml:"company,omitempty"`
}

// Stats represents a set of statistics for an author.
type Stats struct {
	Suspended         bool  `json:"suspended,omitempty" yaml:"suspended,omitempty"`
	CommitsVerified   bool  `json:"verified_commits,omitempty" yaml:"verifiedCommits,omitempty"`
	StrongAuth        bool  `json:"strong_auth,omitempty" yaml:"strongAuth,omitempty"`
	AgeDays           int64 `json:"age_days" yaml:"ageDays"`
	Commits           int64 `json:"commits" yaml:"commits"`
	UnverifiedCommits int64 `json:"unverified_commits" yaml:"unverifiedCommits"`
	PublicRepos       int64 `json:"public_repos,omitempty" yaml:"publicRepos,omitempty"`
	PrivateRepos      int64 `json:"private_repos,omitempty" yaml:"privateRepos,omitempty"`
	Followers         int64 `json:"followers,omitempty" yaml:"followers,omitempty"`
	Following         int64 `json:"following,omitempty" yaml:"following,omitempty"`
	LastCommitDays    int64 `json:"last_commit_days,omitempty" yaml:"lastCommitDays,omitempty"`
	OrgMember         bool  `json:"org_member,omitempty" yaml:"orgMember,omitempty"`
}
