package main

import (
	"github.com/leep-frog/command/sourcerer"
	"github.com/leep-frog/grep"
)

func main() {
	sourcerer.Source(
		grep.HistoryCLI(),
		grep.FilenameCLI(),
		grep.RecursiveCLI(),
		grep.StdinCLI(),
	)
}
