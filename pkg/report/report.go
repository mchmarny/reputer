package report

import (
	"sort"
	"time"
)

// ModelVersion is the current scoring model version.
const ModelVersion = "2.0.0"

// CategoryWeight describes a scoring category and its weight.
type CategoryWeight struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

// Meta holds scoring model metadata.
type Meta struct {
	ModelVersion string           `json:"model_version"`
	Categories   []CategoryWeight `json:"categories"`
}

// Report is the top-level output for a reputation query.
type Report struct {
	Repo              string    `json:"repo,omitempty"`
	AtCommit          string    `json:"at_commit,omitempty"`
	GeneratedOn       time.Time `json:"generated_on,omitempty"`
	TotalCommits      int64     `json:"total_commits,omitempty"`
	TotalContributors int64     `json:"total_contributors,omitempty"`
	Meta              *Meta     `json:"meta,omitempty"`
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
