package github

import (
	"math"
	"time"

	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	ageWeight   = 0.2
	authWeight  = 0.4
	ratioWeight = 0.3
	repoWeight  = 0.1

	n = 10000 // this should be max int32
)

// calculateReputation calculates the reputation score for the given author.
// For demo purposes only, the actual algorithm should be more... sophisticated.
// This calculation is provider specific, this is the GitHub implementation.
// The score is calculated using the following formula:
//
//		score = (normalizedRepositories * repoWeight)
//	       + (normalizedAccountAge * ageWeight)
//	       + (normalizedAuth * authWeight)
//	       + (normalizedRatio * ratioWeight)
//		where:
//		  normalizedRepositories = number of repositories normalized to a range between 0 and 1
//		  normalizedAccountAge = age of the account normalized to a range between 0 and 1
//		  normalizedAuth = 1 / n if the user has two factor auth turned on, 0 otherwise
//		  normalizedRatio = ratio of followers to following normalized to a range between 0 and 1
//
//		The final score is rounded to two decimal places.
func calculateReputation(author *report.Author) error {
	if author == nil {
		return errors.New("author required")
	}

	// if suspended, return 0
	if author.Suspended {
		author.Reputation = 0
		return nil
	}

	// normalize the values to a range between 0 and 1
	normalizedRepositories := float64(author.GetRepos()) / n
	log.Debugf("normalizedRepositories: %f (%d)", normalizedRepositories, author.GetRepos())

	// normalize the age to a range between 0 and 1
	hrs := time.Now().UTC().Sub(author.Created).Hours()
	normalizedAccountAge := hrs / n
	log.Debugf("normalizedAccountAge: %f (%f)", normalizedAccountAge, hrs)

	// if the user has two factor auth turned on, increase the score
	var normalizedAuth float64
	if author.GetTwoFactorAuth() {
		normalizedAuth = 100 / n
		log.Debugf("normalizedAuth: %f", normalizedAuth)
	}

	// if the user has more followers than they are following, increase the score using the ratio
	var normalizedRatio float64
	if author.GetFollowers() > 0 {
		ratio := float64(author.GetFollowing()) / float64(author.GetFollowers())
		normalizedRatio = ratio / n
		log.Debugf("normalizedRatio: %f (%f)", normalizedRatio, ratio)
	}

	// calculate the base reputation score using the weighted average formula
	score := (normalizedRepositories * repoWeight) + (normalizedAccountAge * ageWeight) + (normalizedAuth * authWeight) + (normalizedRatio * ratioWeight)

	author.Reputation = math.Ceil(score*100) / 100
	log.Debugf("score: %.2f", author.Reputation)

	return nil
}
