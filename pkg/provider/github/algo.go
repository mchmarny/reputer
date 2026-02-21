// Package github implements the GitHub reputation provider.
package github

import (
	"math"

	"github.com/mchmarny/reputer/pkg/report"
	log "github.com/sirupsen/logrus"
)

const (
	// weights sum to 1.0
	authWeight           = 0.25
	commitVerifiedWeight = 0.25
	ratioWeight          = 0.15
	ageWeight            = 0.15
	privateRepoWeight    = 0.10
	publicRepoWeight     = 0.10

	// graduated scoring ceilings (signal saturates at 1.0)
	ageFullDays       = 365
	publicRepoFull    = 20
	privateRepoFull   = 10
	followerRatioFull = 10.0
)

// clampedRatio maps val linearly into [0.0, 1.0] with ceil as the saturation point.
func clampedRatio(val, ceil float64) float64 {
	if ceil <= 0 || val <= 0 {
		return 0
	}
	if val >= ceil {
		return 1
	}
	return val / ceil
}

// logCurve maps val into [0.0, 1.0] with logarithmic diminishing returns.
// f(v, c) = log(1+v) / log(1+c), clamped to [0, 1].
func logCurve(val, ceil float64) float64 {
	if ceil <= 0 || val <= 0 {
		return 0
	}
	r := math.Log(1+val) / math.Log(1+ceil)
	if r >= 1 {
		return 1
	}
	return r
}

// expDecay models freshness with a half-life.
// f(v, h) = exp(-v * ln2 / h). Returns 1.0 at v=0, 0.5 at v=halfLife.
func expDecay(val, halfLife float64) float64 {
	if halfLife <= 0 {
		return 0
	}
	if val <= 0 {
		return 1
	}
	return math.Exp(-val * math.Ln2 / halfLife)
}

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

	rep += clampedRatio(float64(s.PrivateRepos), privateRepoFull) * privateRepoWeight
	log.Debugf("private repos [%s]: %.2f (%d)", author.Username, rep, s.PrivateRepos)

	rep += clampedRatio(float64(s.PublicRepos), publicRepoFull) * publicRepoWeight
	log.Debugf("public repos [%s]: %.2f (%d)", author.Username, rep, s.PublicRepos)

	rep += clampedRatio(float64(s.AgeDays), ageFullDays) * ageWeight
	log.Debugf("account age [%s]: %.2f (%d days)", author.Username, rep, s.AgeDays)

	if s.StrongAuth {
		rep += authWeight
	}
	log.Debugf("2fa [%s]: %.2f", author.Username, rep)

	if s.Following > 0 {
		ratio := float64(s.Followers) / float64(s.Following)
		rep += clampedRatio(ratio, followerRatioFull) * ratioWeight
		log.Debugf("follow ratio [%s]: %.2f (%f ratio)", author.Username, rep, ratio)
	}

	if s.Commits > 0 {
		verified := float64(s.Commits-s.UnverifiedCommits) / float64(s.Commits)
		rep += clampedRatio(verified, 1.0) * commitVerifiedWeight
		log.Debugf("commit [%s]: %.2f (%d/%d verified)", author.Username, rep, s.Commits-s.UnverifiedCommits, s.Commits)
	}

	author.Reputation = report.ToFixed(rep, 2)
	log.Debugf("reputation [%s]: %.2f", author.Username, author.Reputation)
}
