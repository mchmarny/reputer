package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mchmarny/reputer/pkg/reputer"
	log "github.com/sirupsen/logrus"
)

var (
	// Set at build time.
	version = "v0.0.1-default"

	repo      string
	commit    string
	file      string
	isDebug   bool
	isVersion bool
)

func init() {
	flag.StringVar(&repo, "repo", "", "Repo URI (required, e.g. github.com/owner/repo)")
	flag.StringVar(&commit, "commit", "", "Commit at which to end the report (optional, inclusive)")
	flag.StringVar(&file, "file", "", "Write output to file at this path (optional, stdout if not specified)")
	flag.BoolVar(&isDebug, "debug", false, "Turns logging verbose (optional, false)")
	flag.BoolVar(&isVersion, "version", false, "Prints version only (optional, false)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s (%s):\n", os.Args[0], version)
		flag.PrintDefaults()
	}
}

func usage() {
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
