package github

import (
	"math"

	"github.com/mchmarny/reputer/pkg/report"
	log "github.com/sirupsen/logrus"
)

const (
	// weight should add up to 1
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
	commitMin      = 1
)

// calculateReputation calculates the reputation score for the given author
// based on the following criteria:
// - suspended users have no reputation
// - repos (private and public)
// - age
// - 2FA
// - follower ratio
// - commit
func calculateReputation(author *report.Author) {
	if author == nil {
		return
	}

	s := author.Stats
	if s == nil {
		log.Debugf("score [%s] - no stats", author.Username)
		return
	}

	// suspended users have no reputation
	if s.Suspended {
		log.Debugf("score [%s] - suspended", author.Username)
		return
	}

	var rep float64

	// repos
	if s.PrivateRepos > privateRepoMin {
		rep += privateRepoWeight
		log.Debugf("private repos [%s]: %.2f (%d)", author.Username, rep, s.PrivateRepos)
	}
	if s.PublicRepos > publicRepoMin {
		rep += publicRepoWeight
		log.Debugf("public repos [%s]: %.2f (%d)", author.Username, rep, s.PublicRepos)
	}

	// age
	// the longer the account has been active the better
	if s.AgeDays > ageDayMin {
		rep += ageWeight
		log.Debugf("account age [%s]: %.2f (%d days)", author.Username, rep, s.AgeDays)
	}

	// 2FA
	// if author has 2FA enabled that means they have a verified email
	if s.StrongAuth {
		rep += authWeight
		log.Debugf("2fa [%s]: %.2f", author.Username, rep)
	}

	// follower ratio
	// not full-proof but a good indicator
	// if author has more followers than they are following
	if s.Following > 0 {
		ratio := float64(s.Followers) / float64(s.Following)
		if ratio > ratioMin {
			rep += ratioWeight
		}
		log.Debugf("follow ratio [%s]: %.2f (%f ratio)", author.Username, rep, ratio)
	}

	// all commits verified
	if s.CommitsVerified {
		rep += commitVerifiedWeight
		log.Debugf("commit [%s]: %.2f (%d commits)", author.Username, rep, s.Commits)
	}

	author.Reputation = toFixed(rep, 2)
	log.Debugf("reputation [%s]: %.2f", author.Username, author.Reputation)
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
