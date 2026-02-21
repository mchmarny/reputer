package github

import (
	"github.com/mchmarny/reputer/pkg/report"
	log "github.com/sirupsen/logrus"
)

const (
	// weights should add up to 0.95
	authWeight           = 0.35
	commitVerifiedWeight = 0.25
	ratioWeight          = 0.15
	ageWeight            = 0.10
	privateRepoWeight    = 0.05
	publicRepoWeight     = 0.05

	privateRepoMin = 1
	publicRepoMin  = 5
	ageDayMin      = 182 // 6 months
	ratioMin       = 1.0
)

// calculateReputation calculates the reputation score for the given author
// based on the following criteria:
// - suspended users have no reputation
// - repos (private and public)
// - age
// - 2FA
// - follower ratio
// - commit verification
func calculateReputation(author *report.Author) {
	if author == nil {
		return
	}

	s := author.Stats
	if s == nil {
		log.Debugf("score [%s] - no stats", author.Username)
		return
	}

	if s.Suspended {
		log.Debugf("score [%s] - suspended", author.Username)
		return
	}

	var rep float64

	if s.PrivateRepos > privateRepoMin {
		rep += privateRepoWeight
		log.Debugf("private repos [%s]: %.2f (%d)", author.Username, rep, s.PrivateRepos)
	}
	if s.PublicRepos > publicRepoMin {
		rep += publicRepoWeight
		log.Debugf("public repos [%s]: %.2f (%d)", author.Username, rep, s.PublicRepos)
	}

	if s.AgeDays > ageDayMin {
		rep += ageWeight
		log.Debugf("account age [%s]: %.2f (%d days)", author.Username, rep, s.AgeDays)
	}

	if s.StrongAuth {
		rep += authWeight
		log.Debugf("2fa [%s]: %.2f", author.Username, rep)
	}

	if s.Following > 0 {
		ratio := float64(s.Followers) / float64(s.Following)
		if ratio > ratioMin {
			rep += ratioWeight
		}
		log.Debugf("follow ratio [%s]: %.2f (%f ratio)", author.Username, rep, ratio)
	}

	if s.CommitsVerified {
		rep += commitVerifiedWeight
		log.Debugf("commit [%s]: %.2f (%d commits)", author.Username, rep, s.Commits)
	}

	author.Reputation = report.ToFixed(rep, 2)
	log.Debugf("reputation [%s]: %.2f", author.Username, author.Reputation)
}
