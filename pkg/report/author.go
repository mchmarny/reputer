package report

import "time"

// Author is a commit author.
type Author struct {
	Login     string    `json:"login,omitempty"`
	Created   time.Time `json:"created,omitempty"`
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
}
