package report

import "time"

type Report struct {
	Repo         string    `json:"repo,omitempty"`
	AtCommit     string    `json:"at_commit,omitempty"`
	GeneratedOn  time.Time `json:"generated_on,omitempty"`
	Contributors []*Author `json:"contributors,omitempty"`
}
