package grep

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/leep-frog/commands/commands"
	"github.com/leep-frog/commands/commandtest"

	"github.com/google/go-cmp/cmp"
)

func TestHistoryLoad(t *testing.T) {
	for _, test := range []struct {
		name    string
		json    string
		want    *Grep
		wantErr string
	}{
		{
			name: "handles empty string",
			want: &Grep{
				inputSource: &history{},
			},
		},
		{
			name:    "handles invalid json",
			json:    "}}",
			wantErr: "failed to unmarshal json for history grep object: invalid character",
			want: &Grep{
				inputSource: &history{},
			},
		},
		{
			name: "handles valid json",
			json: `{"Field": "Value"}`,
			want: &Grep{
				inputSource: &history{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := HistoryGrep()
			err := d.Load(test.json)
			if test.wantErr == "" && err != nil {
				t.Errorf("Load(%s) returned error %v; want nil", test.json, err)
			}
			if test.wantErr != "" && err == nil {
				t.Errorf("Load(%s) returned nil; want err %q", test.json, test.wantErr)
			}
			if test.wantErr != "" && err != nil && !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("Load(%s) returned err %q; want %q", test.json, err.Error(), test.wantErr)
			}

			opts := []cmp.Option{
				cmp.AllowUnexported(history{}),
				cmp.AllowUnexported(Grep{}),
			}
			if diff := cmp.Diff(test.want, d, opts...); diff != "" {
				t.Errorf("Load(%s) produced diff:\n%s", test.json, diff)
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
		want           *commands.WorldState
		wantName       string
		wantStdout     []string
		wantStderr     []string
	}{
		{
			name:       "errors if no option info",
			wantStderr: []string{"OptionInfo is undefined"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue(),
				},
			},
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
			want: &commands.WorldState{
				OptionInfo: &commands.OptionInfo{
					SetupOutputFile: "history.txt",
				},
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue(),
				},
			},
			wantName: "history.txt",
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
			want: &commands.WorldState{
				OptionInfo: &commands.OptionInfo{
					SetupOutputFile: "in/some/path/history.txt",
				},
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^.e"),
				},
			},
			wantName: "in/some/path/history.txt",
			wantStdout: []string{
				fmt.Sprintf("%s%s", matchColor.Format("be"), "ta"),
				fmt.Sprintf("%s%s", matchColor.Format("de"), "lta"),
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
			want: &commands.WorldState{
				OptionInfo: &commands.OptionInfo{
					SetupOutputFile: "in/some/path/history.txt",
				},
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^.*a$"),
				},
				Flags: map[string]*commands.Value{
					caseFlagName: commands.BoolValue(true),
				},
			},
			wantName: "in/some/path/history.txt",
			wantStdout: []string{
				matchColor.Format("alphA"),
				matchColor.Format("beta"),
				matchColor.Format("deltA"),
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
			want: &commands.WorldState{
				OptionInfo: &commands.OptionInfo{
					SetupOutputFile: "history.txt",
				},
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue(),
				},
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
			h := HistoryGrep()
			commandtest.Execute(t, h.Node(), &commands.WorldState{OptionInfo: test.optionInfo, RawArgs: test.args}, test.want, test.wantStdout, test.wantStderr)

			if h.Changed() {
				t.Fatalf("HistoryGrep: Execute(%v, %v) marked Changed as true; want false", h, test.args)
			}

			if test.wantName != gotName {
				t.Fatalf("HistoryGrep: Execute(%v, %v) opened history file %q; want %q", h, test.args, gotName, test.wantName)
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
