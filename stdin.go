package grep

import (
	"bufio"
	"os"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
)

func StdinCLI() *Grep {
	return &Grep{
		InputSource: &stdin{
			scanner: bufio.NewScanner(os.Stdin),
		},
	}
}

type stdin struct {
	scanner *bufio.Scanner
}

func (*stdin) Name() string    { return "ip" }
func (*stdin) Setup() []string { return nil }
func (*stdin) Changed() bool   { return false }
func (*stdin) Flags() []commander.FlagInterface {
	return []commander.FlagInterface{
		beforeFlag,
		afterFlag,
	}
}

func (*stdin) MakeNode(n command.Node) command.Node { return n }

func (si *stdin) Process(output command.Output, data *command.Data, f filter) error {
	list := newLinkedList(f, data, si.scanner)
	for formattedString, _, ok := list.getNext(); ok; formattedString, _, ok = list.getNext() {
		applyFormat(output, data, formattedString)
		output.Stdoutln()
	}

	return nil
}
