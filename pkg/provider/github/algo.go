package github

import (
	"math"
	"time"

	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
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

	hoursInDay = 24

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
func calculateReputation(author *report.Author) error {
	if author == nil {
		return errors.New("author required")
	}

	// suspended users have no reputation
	if author.Suspended {
		log.Debugf("score [%s] - suspended", author.Username)
		return nil
	}

	var rep float64

	// repos
	if author.PrivateRepos > privateRepoMin {
		rep += privateRepoWeight
		log.Debugf("private repos [%s]: %.2f (%d)", author.Username, rep, author.PrivateRepos)
	}
	if author.PublicRepos > publicRepoMin {
		rep += publicRepoWeight
		log.Debugf("public repos [%s]: %.2f (%d)", author.Username, rep, author.PublicRepos)
	}

	// age
	// the longer the account has been active the better
	days := int(math.Ceil(time.Now().UTC().Sub(author.Created).Hours() / hoursInDay))
	if days > ageDayMin {
		rep += ageWeight
		log.Debugf("account age [%s]: %.2f (%d days)", author.Username, rep, days)
	}

	// 2FA
	// if author has 2FA enabled that means they have a verified email
	if author.StrongAuth {
		rep += authWeight
		log.Debugf("2fa [%s]: %.2f", author.Username, rep)
	}

	// follower ratio
	// not full-proof but a good indicator
	// if author has more followers than they are following
	if author.Following > 0 {
		ratio := float64(author.Followers) / float64(author.Following)
		if ratio > ratioMin {
			rep += ratioWeight
		}
		log.Debugf("follow ratio [%s]: %.2f (%f ratio)", author.Username, rep, ratio)
	}

	// all commits verified
	if author.CommitsVerified {
		rep += commitVerifiedWeight
		log.Debugf("commit [%s]: %.2f (%d commits)", author.Username, rep, author.Commits)
	}

	author.Reputation = toFixed(rep, 2)
	log.Debugf("reputation [%s]: %.2f", author.Username, author.Reputation)

	return nil
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
