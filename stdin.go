package grep

import (
	"bufio"
	"os"

	"github.com/leep-frog/command"
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
func (*stdin) Flags() []command.FlagInterface {
	return []command.FlagInterface{
		beforeFlag,
		afterFlag,
	}
}

func (*stdin) MakeNode(n command.Node) command.Node { return n }

func (si *stdin) Process(output command.Output, data *command.Data, f filter) error {
	list := newLinkedList(f, data, si.scanner)
	for formattedString, _, ok := list.getNext(); ok; formattedString, _, ok = list.getNext() {
		output.Stdoutln(formattedString)
	}

	return nil
}
