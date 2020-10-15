package grep

import (
	"bufio"

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
func (*history) Flags() []commands.Flag { return nil }
func (*history) Process(cos commands.CommandOS, args, flags map[string]*commands.Value, oi *commands.OptionInfo, ff filterFunc) (*commands.ExecutorResponse, bool) {
	if oi == nil {
		cos.Stderr("OptionInfo is undefined")
		return nil, false
	}

	f, err := osOpen(oi.SetupOutputFile)
	if err != nil {
		cos.Stderr("failed to open setup output file: %v", err)
		return nil, false
	}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		txt := scanner.Text()
		if ff(txt) {
			cos.Stdout(txt)
		}
	}

	return nil, true
}
