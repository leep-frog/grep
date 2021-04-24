package grep

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
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
			d := HistoryCLI()
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

func TestHistory(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		history    []string
		osOpenErr  error
		want       *command.ExecuteData
		wantData   *command.Data
		wantName   string
		wantStdout []string
		wantStderr []string
		wantErr    error
	}{
		{
			name: "returns history",
			history: []string{
				"alpha",
				"beta",
				"delta",
			},
			wantData: &command.Data{
				Values: map[string]*command.Value{
					patternArgName: command.StringListValue(),
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
			history: []string{
				"alpha",
				"beta",
				"delta",
			},
			wantData: &command.Data{
				Values: map[string]*command.Value{
					patternArgName: command.StringListValue("^.e"),
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
			history: []string{
				"alphA",
				"beta",
				"deltA",
				"zero",
			},
			wantData: &command.Data{
				Values: map[string]*command.Value{
					patternArgName: command.StringListValue("^.*a$"),
					caseFlagName:   command.BoolValue(true),
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
			wantErr: fmt.Errorf("failed to open setup output file: darn"),
			wantData: &command.Data{
				Values: map[string]*command.Value{
					patternArgName: command.StringListValue(),
				},
			},
			wantName: "history.txt",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Stub os.Open if necessary
			if test.osOpenErr != nil {
				oldOpen := osOpen
				osOpen = func(s string) (io.Reader, error) { return nil, test.osOpenErr }
				defer func() { osOpen = oldOpen }()
			}

			// Run test
			h := HistoryCLI()
			setupFile := fakeSetup(t, test.history)
			test.wantData.Values[command.SetupArgName] = command.StringValue(setupFile)
			test.args = append([]string{setupFile}, test.args...)
			command.ExecuteTest(t, command.SerialNodesTo(h.Node(), command.SetupArg), test.args, test.wantErr, test.want, test.wantData, test.wantStdout, test.wantStderr)

			if h.Changed() {
				t.Fatalf("History: Execute(%v, %v) marked Changed as true; want false", h, test.args)
			}
		})
	}
}

// TODO: move this to command package (or create commandtest and put it there).
// TODO: actually make "setup" a field in command.TestObject and just
// do stuff automatically there.
func fakeSetup(t *testing.T, contents []string) string {
	f, err := ioutil.TempFile("", "command_test_setup")
	if err != nil {
		t.Fatalf("ioutil.TempFile('', 'command_test_setup') returned error: %v", err)
	}
	defer f.Close()
	for _, s := range contents {
		fmt.Fprintln(f, s)
	}
	return f.Name()
}

func TestHistoryMetadata(t *testing.T) {
	c := HistoryCLI()

	wantName := "hp"
	if c.Name() != wantName {
		t.Errorf("History.Name() returned %q; want %q", c.Name(), wantName)
	}

	if diff := cmp.Diff([]string{"history"}, c.Setup()); diff != "" {
		t.Errorf("History.Setup() produced diff:\n%s", diff)
	}
}
