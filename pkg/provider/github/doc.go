// Package github implements the GitHub reputation provider.
//
// It fetches commit, user, PR, event, and repository data from the
// GitHub API, delegates reputation scoring to pkg/score, and gathers
// identity, engagement, and behavioral signals including author
// association, PR acceptance rate, cross-repo activity, and fork ratio.
package github
