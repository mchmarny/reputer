// Package github implements the GitHub reputation provider.
//
// It fetches commit and user data from the GitHub API, delegates
// reputation scoring to pkg/score, and checks org membership for
// identity signals.
package github
