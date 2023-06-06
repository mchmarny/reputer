package report

import "fmt"

type Reputation struct {
	Score     float64 `json:"score,omitempty"`
	Algorithm string  `json:"algorithm,omitempty"`
	Version   string  `json:"version,omitempty"`
}

func (r *Reputation) String() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%+v", *r)
}
