package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortAuthors(t *testing.T) {
	r := &Report{
		Contributors: []*Author{
			{Username: "charlie"},
			{Username: "alice"},
			{Username: "bob"},
		},
	}

	r.SortAuthors()

	assert.Equal(t, "alice", r.Contributors[0].Username)
	assert.Equal(t, "bob", r.Contributors[1].Username)
	assert.Equal(t, "charlie", r.Contributors[2].Username)
}

func TestSortAuthorsNilContributors(t *testing.T) {
	r := &Report{Contributors: nil}
	r.SortAuthors() // should not panic
}
