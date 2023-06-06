package report

type Commit struct {
	SHA      string `json:"sha"`
	Verified bool   `json:"verified"`
}
