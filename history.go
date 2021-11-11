package grep

import (
	"bufio"

	"github.com/leep-frog/command"
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

func (*history) Flags() []command.Flag { return nil }
func (*history) Changed() bool         { return false }

func (*history) MakeNode(n *command.Node) *command.Node {
	return command.SerialNodesTo(n, command.SetupArg)
}

func (*history) Process(output command.Output, data *command.Data, f filter) error {
	s, err := osOpen(data.String(command.SetupArgName))
	if err != nil {
		return output.Stderrf("failed to open setup output file: %v", err)
	}

	scanner := bufio.NewScanner(s)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		formattedString, ok := apply(f, scanner.Text(), data)
		if !ok {
			continue
		}
		output.Stdout(formattedString)
	}

	return nil
}
