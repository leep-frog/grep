package grep

import (
	"bufio"
	"encoding/json"
	"fmt"

	"github.com/leep-frog/command"
)

func History() *Grep {
	return &Grep{
		inputSource: &history{},
	}
}

type history struct{}

func (*history) Name() string { return "hp" }
func (*history) Setup() []string {
	return []string{"history"}
}

// Load creates a history grep object from a JSON string.
func (h *history) Load(jsn string) error {
	if jsn == "" {
		h = &history{}
		return nil
	}

	if err := json.Unmarshal([]byte(jsn), h); err != nil {
		return fmt.Errorf("failed to unmarshal json for history grep object: %v", err)
	}
	return nil
}

func (*history) Flags() []command.Flag { return nil }
func (*history) Changed() bool         { return false }

func (*history) Process(output command.Output, data *command.Data, ffs filterFuncs) error {
	f, err := osOpen(data.Values[command.SetupArgName].String())
	if err != nil {
		return output.Stderr("failed to open setup output file: %v", err)
	}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		formattedString, ok := ffs.Apply(scanner.Text())
		if !ok {
			continue
		}
		output.Stdout(formattedString)
	}

	return nil
}
