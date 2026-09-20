package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charlesonunze/nuke/internal/nuke"
	"github.com/charlesonunze/nuke/internal/platform"
)

func main() {
	sys, err := platform.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(nuke.Run(context.Background(), sys, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
