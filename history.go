package grep

import (
	"bufio"
	"encoding/json"
	"fmt"

	"github.com/leep-frog/commands/commands"
)

func HistoryGrep() *Grep {
	return &Grep{
		inputSource: &history{},
	}
}

type history struct{}

func (*history) Name() string  { return "history-grep" }
func (*history) Alias() string { return "hp" }
func (*history) Option() *commands.Option {
	return &commands.Option{
		SetupCommand: "history",
	}
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

func (*history) Flags() []commands.Flag { return nil }
func (*history) Changed() bool          { return false }

//func (*history) Process(cos commands.CommandOS, args, flags map[string]*commands.Value, oi *commands.OptionInfo, ffs filterFuncs) (*commands.ExecutorResponse, bool) {
func (*history) Process(ws *commands.WorldState, ffs filterFuncs) bool {
	if ws.OptionInfo == nil {
		ws.Cos.Stderr("OptionInfo is undefined")
		return false
	}

	f, err := osOpen(ws.OptionInfo.SetupOutputFile)
	if err != nil {
		ws.Cos.Stderr("failed to open setup output file: %v", err)
		return false
	}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		formattedString, ok := ffs.Apply(scanner.Text())
		if !ok {
			continue
		}
		ws.Cos.Stdout(formattedString)
	}

	return true
}
