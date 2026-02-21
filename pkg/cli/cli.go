package cli

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/mchmarny/reputer/pkg/logging"
	"github.com/mchmarny/reputer/pkg/reporter"
)

const usageMsg = `
Usage: reputer [options]

Options:
  --repo     Repo URI (required, e.g. github.com/owner/repo)
  --commit   Commit at which to end the report (optional, inclusive)
  --stats    Includes stats used to calculate reputation (optional)
  --file     Write output to file at this path (optional, stdout if not specified)
  --debug    Turns logging verbose (optional)
  --version  Prints version only (optional)

`

var (
	// Set at build time via ldflags.
	version = "v0.0.1-default"
	commit  = "unknown"
	date    = "unknown"

	repo      string
	commitSHA string
	file      string
	isDebug   bool
	isVersion bool
	withStats bool
)

func init() {
	flag.StringVar(&repo, "repo", "", "")
	flag.StringVar(&commitSHA, "commit", "", "")
	flag.BoolVar(&withStats, "stats", false, "")
	flag.StringVar(&file, "file", "", "")
	flag.BoolVar(&isDebug, "debug", false, "")
	flag.BoolVar(&isVersion, "version", false, "")
}

func usage() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usageMsg)
	}
	flag.Usage()
	os.Exit(1)
}

const appName = "reputer"

func initLogging() {
	logLevel := "info"
	if isDebug {
		logLevel = "debug"
	}

	if isDebug {
		logging.SetDefaultLoggerWithLevel(appName, version, logLevel)
	} else {
		logging.SetDefaultCLILogger(logLevel)
	}
}

// Execute runs the reputer CLI.
func Execute() {
	flag.Parse()
	initLogging()

	if isVersion {
		slog.Info(fmt.Sprintf("Version: %s (commit: %s, date: %s)", version, commit, date))
		os.Exit(0)
	}

	if repo == "" {
		slog.Error("repo is required")
		usage()
	}

	opt := &reporter.ListCommitAuthorsOptions{
		Repo:   repo,
		Commit: commitSHA,
		Stats:  withStats,
		File:   file,
	}

	if err := reporter.ListCommitAuthors(context.Background(), opt); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
