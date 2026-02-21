// Package github implements the GitHub reputation provider.
//
// It fetches commit and user data from the GitHub API, computes
// reputation scores using a risk-weighted categorical model (v2),
// and checks org membership for identity signals.
package github
