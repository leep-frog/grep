package grep

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/leep-frog/command"
)

func StdinCLI() *Grep {
	return &Grep{
		inputSource: &stdin{
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

func (*stdin) PreProcessors() []command.Processor { return nil }

func (s *stdin) Load(jsn string) error {
	if jsn == "" {
		s = &stdin{}
		return nil
	}

	if err := json.Unmarshal([]byte(jsn), s); err != nil {
		return fmt.Errorf("failed to unmarshal json for stdin grep object: %v", err)
	}
	return nil
}

func (si *stdin) Process(output command.Output, data *command.Data, ffs filterFuncs) error {
	list := newLinkedList(ffs, data, si.scanner)
	for formattedString, _, ok := list.getNext(); ok; formattedString, _, ok = list.getNext() {
		output.Stdout(formattedString)
	}

	return nil
}
