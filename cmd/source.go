package main

import (
	"os"

	"github.com/leep-frog/command/sourcerer"
	"github.com/leep-frog/grep"
)

func main() {
	os.Exit(sourcerer.Source([]sourcerer.CLI{
		grep.HistoryCLI(),
		grep.FilenameCLI(),
		grep.RecursiveCLI(),
		grep.StdinCLI(),
	}))
}
