package report

import (
	"fmt"
	"time"
)

// Author is a commit author.
type Author struct {
	Login     string    `json:"login,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	Days      int64     `json:"days,omitempty"`
	Commits   []string  `json:"commits,omitempty"`
	Suspended bool      `json:"suspended,omitempty"`

	Name          *string `json:"name,omitempty"`
	Email         *string `json:"email,omitempty"`
	Company       *string `json:"company,omitempty"`
	Repos         *int    `json:"repos,omitempty"`
	Gists         *int    `json:"gists,omitempty"`
	Followers     *int    `json:"followers,omitempty"`
	Following     *int    `json:"following,omitempty"`
	TwoFactorAuth *bool   `json:"two_factor_auth,omitempty"`

	Reputation float64 `json:"reputation,omitempty"`
}

// GetName returns the name of the author.
func (a *Author) GetName() string {
	if a == nil || a.Name == nil {
		return ""
	}
	return *a.Name
}

// GetEmail returns the email of the author.
func (a *Author) GetEmail() string {
	if a == nil || a.Email == nil {
		return ""
	}
	return *a.Email
}

// GetCompany returns the company of the author.
func (a *Author) GetCompany() string {
	if a == nil || a.Company == nil {
		return ""
	}
	return *a.Company
}

// GetRepos returns the number of repos of the author.
func (a *Author) GetRepos() int {
	if a == nil || a.Repos == nil {
		return 0
	}
	return *a.Repos
}

// GetGists returns the number of gists of the author.
func (a *Author) GetGists() int {
	if a == nil || a.Gists == nil {
		return 0
	}
	return *a.Gists
}

// GetFollowers returns the number of followers of the author.
func (a *Author) GetFollowers() int {
	if a == nil || a.Followers == nil {
		return 0
	}
	return *a.Followers
}

// GetFollowing returns the number of following of the author.
func (a *Author) GetFollowing() int {
	if a == nil || a.Following == nil {
		return 0
	}
	return *a.Following
}

// GetTwoFactorAuth returns the two factor auth status of the author.
func (a *Author) GetTwoFactorAuth() bool {
	if a == nil || a.TwoFactorAuth == nil {
		return false
	}
	return *a.TwoFactorAuth
}

func (a *Author) String() string {
	return fmt.Sprintf("Author: %s, Commits: %d, Suspended: %t, Name: %s, Email: %s, Company: %s, Repos: %d, Gists: %d, Followers: %d, Following: %d, TwoFactorAuth: %t", a.Login, len(a.Commits), a.Suspended, a.GetName(), a.GetEmail(), a.GetCompany(), a.GetRepos(), a.GetGists(), a.GetFollowers(), a.GetFollowing(), a.GetTwoFactorAuth())
}
