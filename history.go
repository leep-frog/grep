package grep

import (
	"bufio"
	"strings"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
)

func HistoryCLI() *Grep {
	return &Grep{
		InputSource: &history{},
	}
}

type history struct{}

func (*history) Name() string { return "hp" }
func (*history) Setup() []string {
	return []string{"history"}
}

func (*history) Flags() []commander.FlagInterface { return nil }
func (*history) Changed() bool                    { return false }

func (*history) MakeNode(n command.Node) command.Node {
	return n
}

func (*history) Process(output command.Output, data *command.Data, f filter, ss *sliceSet) error {
	s, err := osOpen(commander.SetupOutputFile(data))
	if err != nil {
		return output.Stderrf("failed to open setup output file: %v\n", err)
	}

	scanner := bufio.NewScanner(s)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		// We need to replace all null characters because (for windows)
		// null characters creep into the history output file for some reason.
		formattedString, ok := apply(f, strings.ReplaceAll(scanner.Text(), "\x00", ""), data, ss)
		if !ok {
			continue
		}
		applyFormat(output, data, formattedString)
		output.Stdoutln()
	}

	return nil
}
