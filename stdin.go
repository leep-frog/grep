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
func (*stdin) Flags() []command.Flag {
	return []command.Flag{
		beforeFlag,
		afterFlag,
	}
}

func (*stdin) MakeNode(n *command.Node) *command.Node { return n }

func (si *stdin) Process(output command.Output, data *command.Data, ffs filterFuncs) error {
	list := newLinkedList(ffs, data, si.scanner)
	for formattedString, _, ok := list.getNext(); ok; formattedString, _, ok = list.getNext() {
		output.Stdout(formattedString)
	}

	return nil
}
