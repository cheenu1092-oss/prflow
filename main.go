package main

import (
	"fmt"
	"os"

	"github.com/cheenu1092-oss/prflow/cmd"
)

// Set via ldflags at build time.
var version, commit, date string

func main() {
	cmd.Version = version
	cmd.Commit = commit
	cmd.Date = date

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
