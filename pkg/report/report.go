package report

import (
	"sort"
	"time"
)

type Report struct {
	Repo              string    `json:"repo,omitempty"`
	AtCommit          string    `json:"at_commit,omitempty"`
	GeneratedOn       time.Time `json:"generated_on,omitempty"`
	TotalCommits      int64     `json:"total_commits,omitempty"`
	TotalContributors int64     `json:"total_contributors,omitempty"`
	Contributors      []*Author `json:"contributors,omitempty"`
}

// SortAuthors sorts the authors by username.
func (r *Report) SortAuthors() {
	if r.Contributors == nil {
		return
	}

	sort.Slice(r.Contributors, func(i, j int) bool {
		return r.Contributors[i].Username < r.Contributors[j].Username
	})
}
