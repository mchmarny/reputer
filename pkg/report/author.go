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
	AgeDays           int64 `json:"age_days" yaml:"ageDays"`
	Commits           int64 `json:"commits" yaml:"commits"`
	UnverifiedCommits int64 `json:"unverified_commits" yaml:"unverifiedCommits"`
	PublicRepos       int64 `json:"public_repos,omitempty" yaml:"publicRepos,omitempty"`
	Followers         int64 `json:"followers,omitempty" yaml:"followers,omitempty"`
	Following         int64 `json:"following,omitempty" yaml:"following,omitempty"`
	LastCommitDays    int64 `json:"last_commit_days,omitempty" yaml:"lastCommitDays,omitempty"`
	OrgMember         bool  `json:"org_member,omitempty" yaml:"orgMember,omitempty"`

	// v3 fields
	AuthorAssociation string `json:"author_association,omitempty" yaml:"authorAssociation,omitempty"`
	HasBio            bool   `json:"has_bio,omitempty" yaml:"hasBio,omitempty"`
	HasCompany        bool   `json:"has_company,omitempty" yaml:"hasCompany,omitempty"`
	HasLocation       bool   `json:"has_location,omitempty" yaml:"hasLocation,omitempty"`
	HasWebsite        bool   `json:"has_website,omitempty" yaml:"hasWebsite,omitempty"`
	PRsMerged         int64  `json:"prs_merged,omitempty" yaml:"prsMerged,omitempty"`
	PRsClosed         int64  `json:"prs_closed,omitempty" yaml:"prsClosed,omitempty"`
	RecentPRRepoCount int64  `json:"recent_pr_repo_count,omitempty" yaml:"recentPRRepoCount,omitempty"`
	ForkedRepos       int64  `json:"forked_repos,omitempty" yaml:"forkedRepos,omitempty"`
}
