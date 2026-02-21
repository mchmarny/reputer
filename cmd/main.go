// Package main is the entry point for the reputer CLI.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mchmarny/reputer/pkg/reputer"
	log "github.com/sirupsen/logrus"
)

const (
	usageMsg = `
Usage: reputer [options]

Options:
  --repo     Repo URI (required, e.g. github.com/owner/repo)
  --commit   Commit at which to end the report (optional, inclusive)
  --stats    Includes stats used to calculate reputation (optional)
  --file     Write output to file at this path (optional, stdout if not specified)
  --debug    Turns logging verbose (optional)
  --version  Prints version only (optional)

`
)

var (
	// Set at build time.
	version = "v0.0.1-default"

	repo      string
	commit    string
	file      string
	isDebug   bool
	isVersion bool
	withStats bool
)

func init() {
	flag.StringVar(&repo, "repo", "", "")
	flag.StringVar(&commit, "commit", "", "")
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

func main() {
	flag.Parse()
	initLogging()

	if isVersion {
		log.Infof("Version: %s", version)
		os.Exit(0)
	}

	if repo == "" {
		log.Error("repo is required")
		usage()
	}

	opt := &reputer.ListCommitAuthorsOptions{
		Repo:   repo,
		Commit: commit,
		Stats:  withStats,
		File:   file,
	}

	if err := reputer.ListCommitAuthors(context.Background(), opt); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func initLogging() {
	log.SetOutput(os.Stderr)
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:          false,
		ForceColors:            true,
		DisableTimestamp:       true,
		PadLevelText:           false,
		DisableLevelTruncation: true,
	})

	if isDebug {
		log.SetLevel(log.DebugLevel)
	}
}
