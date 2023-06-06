package report

import "time"

type Report struct {
	Repo              string    `json:"repo,omitempty"`
	AtCommit          string    `json:"at_commit,omitempty"`
	GeneratedOn       time.Time `json:"generated_on,omitempty"`
	TotalCommits      int64     `json:"total_commits,omitempty"`
	TotalContributors int64     `json:"total_contributors,omitempty"`
	Contributors      []*Author `json:"contributors,omitempty"`
}
