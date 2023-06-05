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

	repo   string
	commit string
	debug  bool
)

func init() {
	flag.StringVar(&repo, "repo", "", "Repo URI (e.g. github.com/owner/repo)")
	flag.StringVar(&commit, "commit", "", "Commit from which to start the report (optional, inclusive)")
	flag.BoolVar(&debug, "debug", false, "Turns logging verbose (optional, false)")
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

	if repo == "" {
		usage()
	}

	opt := &reputer.ListCommitAuthorsOptions{
		Repo:   repo,
		Commit: commit,
	}

	if err := reputer.ListCommitAuthors(context.Background(), opt); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func initLogging() {
	log.SetOutput(os.Stderr)
	log.SetLevel(log.InfoLevel)

	if debug {
		log.SetLevel(log.DebugLevel)
	}
}
