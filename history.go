package grep

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/leep-frog/cli/cli"
	"github.com/leep-frog/cli/commands"
)

var (
	cmdRun = internalCmdRun
	goos   = internalGOOS
)

func HistoryGrep() cli.CLI {
	return &grep{
		inputSource: &history{},
	}
}

type history struct{}

func (*history) Name() string           { return "history-grep" }
func (*history) Alias() string          { return "hp" }
func (*history) Flags() []commands.Flag { return nil }
func (*history) Process(args, flags map[string]*commands.Value, ff filterFunc) ([]string, error) {
	if goos() == "windows" {
		return execute("doskey", []string{"/history"}, ff)
	}
	return execute("history", nil, ff)
}

// separate function so it can be stubbed out for tests
func execute(command string, args []string, ff filterFunc) ([]string, error) {
	stdout := &strings.Builder{}
	cmd := &exec.Cmd{
		Path:   command,
		Args:   args,
		Stdout: stdout,
	}
	if err := cmdRun(cmd); err != nil {
		return nil, fmt.Errorf("failed to run history command: %v", err)
	}

	ss := strings.Split(stdout.String(), "\n")
	lines := make([]string, 0, len(ss))
	for _, s := range ss {
		if ff(s) {
			lines = append(lines, s)
		}
	}
	return lines, nil
}

func internalCmdRun(cmd *exec.Cmd) error {
	return cmd.Run()
}

func internalGOOS() string {
	return runtime.GOOS
}
