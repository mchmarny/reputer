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
	ageWeight         = 0.2
	authWeight        = 0.3
	ratioWeight       = 0.2
	privateRepoWeight = 0.1
	publicRepoWeight  = 0.1
	commitWeight      = 0.1

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
		log.Debugf("[%s] score - suspended", author.Username)
		return nil
	}

	var rep float64

	// repos
	if author.PrivateRepos > privateRepoMin {
		rep += privateRepoWeight
		log.Debugf("[%s] score - private repos: %.1f (%d)", author.Username, rep, author.PrivateRepos)
	}
	if author.PublicRepos > publicRepoMin {
		rep += publicRepoWeight
		log.Debugf("[%s] score - public repos: %.1f (%d)", author.Username, rep, author.PublicRepos)
	}

	// age
	// the longer the account has been active the better
	days := int(math.Ceil(time.Now().UTC().Sub(author.Created).Hours() / hoursInDay))
	if days > ageDayMin {
		rep += ageWeight
		log.Debugf("[%s] score - age: %.1f (%d days)", author.Username, rep, days)
	}

	// 2FA
	// if author has 2FA enabled that means they have a verified email
	if author.TwoFactorAuth {
		rep += authWeight
		log.Debugf("[%s] score - 2FA: %.1f", author.Username, rep)
	}

	// follower ratio
	// not full-proof but a good indicator
	// if author has more followers than they are following
	if author.Following > 0 {
		ratio := float64(author.Followers) / float64(author.Following)
		if ratio > ratioMin {
			rep += ratioWeight
		}
		log.Debugf("[%s] score - ratio: %.1f (%f ratio)", author.Username, rep, ratio)
	}

	// commit
	// 1 commit is not enough (implicit by being in the list of authors)
	// multiple commits indicate additional opportunities for review
	if len(author.Commits) > commitMin {
		rep += commitWeight
		log.Debugf("[%s] score - commit: %.1f (%d commits)", author.Username, rep, len(author.Commits))
	}

	author.Reputation = toFixed(rep, 1)
	log.Debugf("[%s] score: %.1f", author.Username, author.Reputation)

	return nil
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
