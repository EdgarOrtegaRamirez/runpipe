package main

import (
	"os"

	"github.com/EdgarOrtegaRamirez/runpipe/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
