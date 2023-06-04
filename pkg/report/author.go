package report

// Author is a commit author.
type Author struct {
	Login string `json:"login,omitempty"`

	Name    *string `json:"name,omitempty"`
	Email   *string `json:"email,omitempty"`
	Company *string `json:"company,omitempty"`

	Commits []string `json:"commits,omitempty"`
}
