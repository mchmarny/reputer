package github

import (
	"math"

	"github.com/mchmarny/reputer/pkg/report"
	log "github.com/sirupsen/logrus"
)

const (
	// Category weights (sum to 1.0).
	// Derived from threat-model ranking: Provenance > Identity > Engagement > Community.
	provenanceWeight = 0.35
	ageWeight        = 0.15
	orgMemberWeight  = 0.10
	proportionWeight = 0.15
	recencyWeight    = 0.10
	followerWeight   = 0.10
	repoCountWeight  = 0.05

	// Ceilings and parameters.
	ageCeilDays           = 730
	followerRatioCeil     = 10.0
	repoCountCeil         = 30.0
	baseHalfLifeDays      = 90.0
	minHalfLifeMultiple   = 0.25
	provenanceFloor       = 0.1
	minProportionCeil     = 0.05
	minConfidenceCommits  = 30
	confCommitsPerContrib = 10
	noTFAMaxDiscount      = 0.5
)

// Exported category weights derived from signal constants above.
var (
	CategoryProvenanceWeight = provenanceWeight
	CategoryIdentityWeight   = ageWeight + orgMemberWeight
	CategoryEngagementWeight = proportionWeight + recencyWeight
	CategoryCommunityWeight  = followerWeight + repoCountWeight
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
func expDecay(val, halfLife float64) float64 {
	if halfLife <= 0 {
		return 0
	}
	if val <= 0 {
		return 1
	}
	return math.Exp(-val * math.Ln2 / halfLife)
}

// calculateReputation scores an author using the v2 risk-weighted categorical model.
func calculateReputation(author *report.Author, totalCommits int64, totalContributors int) {
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

	// --- Category 1: Code Provenance (0.35) ---
	if s.Commits > 0 && totalCommits > 0 {
		verifiedRatio := float64(s.Commits-s.UnverifiedCommits) / float64(s.Commits)
		proportion := float64(s.Commits) / float64(totalCommits)
		propCeil := math.Max(1.0/float64(max(totalContributors, 1)), minProportionCeil)

		securityMultiplier := 1.0
		if !s.StrongAuth {
			securityMultiplier = 1.0 - noTFAMaxDiscount*clampedRatio(proportion, propCeil)
		}

		rawScore := verifiedRatio * securityMultiplier
		if s.StrongAuth && rawScore < provenanceFloor {
			rawScore = provenanceFloor
		}

		rep += rawScore * provenanceWeight
		log.Debugf("provenance [%s]: %.4f (verified=%.2f, multiplier=%.2f)",
			author.Username, rep, verifiedRatio, securityMultiplier)
	} else if s.StrongAuth {
		rep += provenanceFloor * provenanceWeight
		log.Debugf("provenance [%s]: %.4f (2FA floor, no commits)",
			author.Username, rep)
	}

	// --- Category 2: Identity Authenticity (0.25) ---
	rep += logCurve(float64(s.AgeDays), ageCeilDays) * ageWeight
	log.Debugf("age [%s]: %.4f (%d days)", author.Username, rep, s.AgeDays)

	if s.OrgMember {
		rep += orgMemberWeight
	}
	log.Debugf("org [%s]: %.4f (member=%v)", author.Username, rep, s.OrgMember)

	// --- Category 3: Engagement Depth (0.25) ---
	if s.Commits > 0 && totalCommits > 0 {
		proportion := float64(s.Commits) / float64(totalCommits)
		propCeil := math.Max(1.0/float64(max(totalContributors, 1)), minProportionCeil)

		confThreshold := float64(max(
			int64(totalContributors)*int64(confCommitsPerContrib),
			int64(minConfidenceCommits),
		))
		confidence := math.Min(float64(totalCommits)/confThreshold, 1.0)

		rep += clampedRatio(proportion, propCeil) * confidence * proportionWeight
		log.Debugf("proportion [%s]: %.4f (prop=%.3f, ceil=%.3f, conf=%.3f)",
			author.Username, rep, proportion, propCeil, confidence)
	}

	numContrib := max(totalContributors, 1)
	halfLifeMult := math.Max(1.0/math.Log(1+float64(numContrib)), minHalfLifeMultiple)
	if halfLifeMult > 1.0 {
		halfLifeMult = 1.0
	}
	halfLife := baseHalfLifeDays * halfLifeMult
	rep += expDecay(float64(s.LastCommitDays), halfLife) * recencyWeight
	log.Debugf("recency [%s]: %.4f (%d days, halfLife=%.1f)",
		author.Username, rep, s.LastCommitDays, halfLife)

	// --- Category 4: Community Standing (0.15) ---
	if s.Following > 0 {
		ratio := float64(s.Followers) / float64(s.Following)
		rep += logCurve(ratio, followerRatioCeil) * followerWeight
		log.Debugf("followers [%s]: %.4f (ratio=%.2f)", author.Username, rep, ratio)
	}

	totalRepos := float64(s.PublicRepos + s.PrivateRepos)
	rep += logCurve(totalRepos, repoCountCeil) * repoCountWeight
	log.Debugf("repos [%s]: %.4f (%d combined)", author.Username, rep, int64(totalRepos))

	author.Reputation = report.ToFixed(rep, 2)
	log.Debugf("reputation [%s]: %.2f", author.Username, author.Reputation)
}
