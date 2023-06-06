package report

import (
	"fmt"
	"time"
)

// Author represents a commit author.
type Author struct {
	Username      string                 `json:"username,omitempty"`
	Created       time.Time              `json:"created,omitempty"`
	Suspended     bool                   `json:"suspended,omitempty"`
	PublicRepos   int64                  `json:"public_repos,omitempty"`
	PrivateRepos  int64                  `json:"private_repos,omitempty"`
	Followers     int64                  `json:"followers,omitempty"`
	Following     int64                  `json:"following,omitempty"`
	TwoFactorAuth bool                   `json:"two_factor_auth,omitempty"`
	Reputation    Reputation             `json:"reputation,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
	Commits       []*Commit              `json:"commits,omitempty"`
}

func (a *Author) String() string {
	if a == nil {
		return ""
	}
	return fmt.Sprintf("%+v", *a)
}
