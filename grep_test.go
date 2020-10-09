package grep

import (
	"fmt"
	"io"
	"os/exec"
	"testing"

	"github.com/leep-frog/cli/commands"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/cli/cli"
)

func TestLoad(t *testing.T) {
	for _, test := range []struct {
		name string
		json string
		want *grep
	}{
		{
			name: "handles empty string",
		},
		{
			name: "handles invalid json",
			json: "}}",
		},
		{
			name: "handles valid json",
			json: `{"Field": "Value"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := &grep{}
			if err := d.Load(test.json); err != nil {
				t.Fatalf("Load(%v) should return nil; got %v", test.json, err)
			}
		})
	}
}

func TestHistoryGrep(t *testing.T) {
	for _, test := range []struct {
		name        string
		args        []string
		cmdRunErr   error
		stdout      string
		goos        string
		wantResp    *commands.ExecutorResponse
		wantCommand string
		wantArgs    []string
	}{
		{
			name:        "returns history",
			goos:        "windows",
			stdout:      "alpha\nbeta\ndelta",
			wantCommand: "doskey",
			wantArgs:    []string{"/history"},
			wantResp: &commands.ExecutorResponse{
				Stdout: []string{
					"alpha",
					"beta",
					"delta",
				},
			},
		},
		{
			name:        "returns history on linux",
			goos:        "linux",
			stdout:      "alpha\nbeta\ndelta",
			wantCommand: "history",
			wantResp: &commands.ExecutorResponse{
				Stdout: []string{
					"alpha",
					"beta",
					"delta",
				},
			},
		},
		{
			name:        "filters history",
			args:        []string{"^.e"},
			goos:        "windows",
			stdout:      "alpha\nbeta\ndelta",
			wantCommand: "doskey",
			wantArgs:    []string{"/history"},
			wantResp: &commands.ExecutorResponse{
				Stdout: []string{
					"beta",
					"delta",
				},
			},
		},
		{
			name:        "errors on cmd run error",
			cmdRunErr:   fmt.Errorf("darn"),
			goos:        "windows",
			wantCommand: "doskey",
			wantArgs:    []string{"/history"},
			wantResp: &commands.ExecutorResponse{
				Stderr: []string{
					"failed to run history command: darn",
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Mock cmd.Run()
			var gotCommand string
			var gotArgs []string
			oldRun := cmdRun
			cmdRun = func(cmd *exec.Cmd) error {
				gotCommand = cmd.Path
				gotArgs = cmd.Args
				_, err := io.WriteString(cmd.Stdout, test.stdout)
				if err != nil {
					t.Fatalf("failed to mock write to stdout: %v", err)
				}
				return test.cmdRunErr
			}
			defer func() { cmdRun = oldRun }()

			// Mock goos
			if test.goos != "" {
				oldGOOS := goos
				goos = func() string { return test.goos }
				defer func() { goos = oldGOOS }()
			}

			// Run test
			c := HistoryGrep()

			got, err := cli.Execute(c, test.args)
			if err != nil {
				t.Fatalf("RecursiveGrep: Execute(%v, %v) returned error (%v); want nil", c, test.args, err)
			}
			if diff := cmp.Diff(test.wantResp, got); diff != "" {
				t.Fatalf("RecursiveGrep: Execute(%v, %v) produced response diff (-want, +got):\n%s", c, test.args, diff)
			}
			if c.Changed() {
				t.Fatalf("RecursiveGrep: Execute(%v, %v) marked Changed as true; want false", c, test.args)
			}

			if gotCommand != test.wantCommand {
				t.Fatalf("RecursiveGrep: Execute(%v, %v) ran command %q; want %q", c, test.args, gotCommand, test.wantCommand)
			}

			if diff := cmp.Diff(test.wantArgs, gotArgs); diff != "" {
				t.Fatalf("HistoryGrep: Execute(%v, %v) produced args diff:\n%s", c, test.args, diff)
			}
		})
	}
}

func TestHistoryMetadata(t *testing.T) {
	c := HistoryGrep()

	wantName := "history-grep"
	if c.Name() != wantName {
		t.Errorf("HistoryGrep.Name returned %q; want %q", c.Name(), wantName)
	}

	wantAlias := "hp"
	if c.Alias() != wantAlias {
		t.Errorf("HistoryGrep.Alias returned %q; want %q", c.Alias(), wantAlias)
	}
}
