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

	repo       string
	fromCommit string
	atCommit   string
	debug      bool
)

func init() {
	flag.StringVar(&repo, "repo", "", "Repo URI (required, e.g. github.com/owner/repo)")
	flag.StringVar(&atCommit, "commit", "", "Commit at which end the report (required, inclusive)")
	flag.StringVar(&fromCommit, "from", "", "Commit from which to start the report (optional, inclusive)")
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

	if repo == "" || atCommit == "" {
		usage()
	}

	opt := &reputer.ListCommitAuthorsOptions{
		Repo:       repo,
		FromCommit: fromCommit,
		AtCommit:   atCommit,
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
