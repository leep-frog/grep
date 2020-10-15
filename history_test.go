package grep

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/leep-frog/commands/commands"

	"github.com/google/go-cmp/cmp"
)

func TestLoad(t *testing.T) {
	for _, test := range []struct {
		name string
		json string
		want *Grep
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
			d := &Grep{}
			if err := d.Load(test.json); err != nil {
				t.Fatalf("Load(%v) should return nil; got %v", test.json, err)
			}
		})
	}
}

func TestHistoryGrep(t *testing.T) {
	for _, test := range []struct {
		name           string
		args           []string
		optionInfo     *commands.OptionInfo
		osOpenErr      error
		osOpenContents []string
		wantOK         bool
		wantResp       *commands.ExecutorResponse
		wantName       string
		wantStdout     []string
		wantStderr     []string
	}{
		{
			name:       "errors if no option info",
			wantStderr: []string{"OptionInfo is undefined"},
		},
		{
			name: "returns history",
			osOpenContents: []string{
				"alpha",
				"beta",
				"delta",
			},
			optionInfo: &commands.OptionInfo{
				SetupOutputFile: "history.txt",
			},
			wantName: "history.txt",
			wantOK:   true,
			wantStdout: []string{
				"alpha",
				"beta",
				"delta",
			},
		},
		{
			name: "filters history",
			args: []string{"^.e"},
			osOpenContents: []string{
				"alpha",
				"beta",
				"delta",
			},
			optionInfo: &commands.OptionInfo{
				SetupOutputFile: "in/some/path/history.txt",
			},
			wantName: "in/some/path/history.txt",
			wantOK:   true,
			wantStdout: []string{
				"beta",
				"delta",
			},
		},
		{
			name: "filters history ignoring case",
			args: []string{"^.*a$", "-i"},
			osOpenContents: []string{
				"alphA",
				"beta",
				"deltA",
				"zero",
			},
			optionInfo: &commands.OptionInfo{
				SetupOutputFile: "in/some/path/history.txt",
			},
			wantName: "in/some/path/history.txt",
			wantOK:   true,
			wantStdout: []string{
				"alphA",
				"beta",
				"deltA",
			},
		},
		{
			name:      "errors on os.Open error",
			osOpenErr: fmt.Errorf("darn"),
			wantStderr: []string{
				"failed to open setup output file: darn",
			},
			optionInfo: &commands.OptionInfo{
				SetupOutputFile: "history.txt",
			},
			wantName: "history.txt",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var gotName string
			oldOpen := osOpen
			osOpen = func(name string) (io.Reader, error) {
				gotName = name
				return strings.NewReader(strings.Join(test.osOpenContents, "\n")), test.osOpenErr
			}
			defer func() { osOpen = oldOpen }()

			// Run test
			tcos := &commands.TestCommandOS{}
			c := HistoryGrep()
			got, ok := commands.Execute(tcos, c.Command(), test.args, test.optionInfo)
			if ok != test.wantOK {
				t.Fatalf("HistoryGrep: commands.Execute(%v) returned %v for ok; want %v", test.args, ok, test.wantOK)
			}
			if diff := cmp.Diff(test.wantResp, got); diff != "" {
				t.Fatalf("HistoryGrep: Execute(%v, %v) produced response diff (-want, +got):\n%s", c, test.args, diff)
			}

			if diff := cmp.Diff(test.wantStdout, tcos.GetStdout()); diff != "" {
				t.Errorf("HistoryGrep: command.Execute(%v) produced stdout diff (-want, +got):\n%s", test.args, diff)
			}
			if diff := cmp.Diff(test.wantStderr, tcos.GetStderr()); diff != "" {
				t.Errorf("HistoryGrep: command.Execute(%v) produced stderr diff (-want, +got):\n%s", test.args, diff)
			}

			if c.Changed() {
				t.Fatalf("HistoryGrep: Execute(%v, %v) marked Changed as true; want false", c, test.args)
			}

			if test.wantName != gotName {
				t.Fatalf("HistoryGrep: Execute(%v, %v) opened history file %q; want %q", c, test.args, gotName, test.wantName)
			}
		})
	}
}

func TestHistoryMetadata(t *testing.T) {
	c := HistoryGrep()

	wantName := "history-grep"
	if c.Name() != wantName {
		t.Errorf("HistoryGrep.Name() returned %q; want %q", c.Name(), wantName)
	}

	wantAlias := "hp"
	if c.Alias() != wantAlias {
		t.Errorf("HistoryGrep.Alias() returned %q; want %q", c.Alias(), wantAlias)
	}

	wantOption := &commands.Option{SetupCommand: "history"}
	if diff := cmp.Diff(wantOption, c.Option()); diff != "" {
		t.Errorf("HistoryGrep.Option() produced diff:\n%s", diff)
	}
}
