package grep

import (
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
func (*history) Process(cos commands.CommandOS, args, flags map[string]*commands.Value, ff filterFunc) (*commands.ExecutorResponse, bool) {
	if goos() == "windows" {
		return execute(cos, "doskey", []string{"/history"}, ff)
	}
	return execute(cos, "history", nil, ff)
}

// separate function so it can be stubbed out for tests
func execute(cos commands.CommandOS, command string, args []string, ff filterFunc) (*commands.ExecutorResponse, bool) {
	stdout := &strings.Builder{}
	cmd := &exec.Cmd{
		Path:   command,
		Args:   args,
		Stdout: stdout,
	}
	if err := cmdRun(cmd); err != nil {
		cos.Stderr("failed to run history command: %v", err)
		return nil, false
	}

	ss := strings.Split(stdout.String(), "\n")
	for _, s := range ss {
		if ff(s) {
			cos.Stdout(s)
		}
	}
	return &commands.ExecutorResponse{}, true
}

func internalCmdRun(cmd *exec.Cmd) error {
	return cmd.Run()
}

func internalGOOS() string {
	return runtime.GOOS
}
