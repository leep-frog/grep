package grep

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
)

func TestHistoryLoad(t *testing.T) {
	for _, test := range []struct {
		name    string
		json    string
		want    *Grep
		WantErr string
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
			WantErr: "failed to unmarshal json for history grep object: invalid character",
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
			if test.WantErr == "" && err != nil {
				t.Errorf("Load(%s) returned error %v; want nil", test.json, err)
			}
			if test.WantErr != "" && err == nil {
				t.Errorf("Load(%s) returned nil; want err %q", test.json, test.WantErr)
			}
			if test.WantErr != "" && err != nil && !strings.Contains(err.Error(), test.WantErr) {
				t.Errorf("Load(%s) returned err %q; want %q", test.json, err.Error(), test.WantErr)
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
		name      string
		history   []string
		osOpenErr error
		etc       *command.ExecuteTestCase
	}{
		{
			name: "returns history",
			history: []string{
				"alpha",
				"beta",
				"delta",
			},
			etc: &command.ExecuteTestCase{
				WantStdout: []string{
					"alpha",
					"beta",
					"delta",
				},
			},
		},
		{
			name: "filters history",
			history: []string{
				"alpha",
				"beta",
				"delta",
			},
			etc: &command.ExecuteTestCase{
				Args: []string{"^.e"},
				WantData: &command.Data{
					Values: map[string]*command.Value{
						patternArgName: command.StringListValue("^.e"),
					},
				},
				WantStdout: []string{
					fmt.Sprintf("%s%s", matchColor.Format("be"), "ta"),
					fmt.Sprintf("%s%s", matchColor.Format("de"), "lta"),
				},
			},
		},
		{
			name: "filters history ignoring case",
			history: []string{
				"alphA",
				"beta",
				"deltA",
				"zero",
			},
			etc: &command.ExecuteTestCase{
				Args: []string{"^.*a$", "-i"},
				WantData: &command.Data{
					Values: map[string]*command.Value{
						patternArgName:  command.StringListValue("^.*a$"),
						caseFlag.Name(): command.BoolValue(true),
					},
				},
				WantStdout: []string{
					matchColor.Format("alphA"),
					matchColor.Format("beta"),
					matchColor.Format("deltA"),
				},
			},
		},
		{
			name:      "errors on os.Open error",
			osOpenErr: fmt.Errorf("darn"),
			etc: &command.ExecuteTestCase{
				WantStderr: []string{
					"failed to open setup output file: darn",
				},
				WantErr: fmt.Errorf("failed to open setup output file: darn"),
			},
		},
		{
			name: "works with match only",
			history: []string{
				"qwerTyuiop",
				"asdTfghjTkl",
				"TxcvbnmT",
			},
			etc: &command.ExecuteTestCase{
				Args: []string{"T.*T", "-o"},
				WantData: &command.Data{
					Values: map[string]*command.Value{
						patternArgName:       command.StringListValue("T.*T"),
						matchOnlyFlag.Name(): command.BoolValue(true),
					},
				},
				WantStdout: []string{
					"TfghjT",
					"TxcvbnmT",
				},
			},
		},
		{
			name: "works with match only and overlapping matches",
			history: []string{
				"qwerTyuiop",
				"aSdTfghSjTkl",
				"TxScvbSnmT",
			},
			etc: &command.ExecuteTestCase{
				Args: []string{"T.*T", "S.*S", "-o"},
				WantData: &command.Data{
					Values: map[string]*command.Value{
						patternArgName:       command.StringListValue("T.*T", "S.*S"),
						matchOnlyFlag.Name(): command.BoolValue(true),
					},
				},
				WantStdout: []string{
					"SdTfghSjT",
					"TxScvbSnmT",
				},
			},
		},
		{
			name: "works with match only and non-overlapping matches",
			history: []string{
				"qwerTyuiop",
				"SaSdfTghjTkl",
				"TzTxcvbSnmS",
			},
			etc: &command.ExecuteTestCase{
				Args: []string{"T.*T", "S.*S", "-o"},
				WantData: &command.Data{
					Values: map[string]*command.Value{
						patternArgName:       command.StringListValue("T.*T", "S.*S"),
						matchOnlyFlag.Name(): command.BoolValue(true),
					},
				},
				WantStdout: []string{
					"SaS...TghjT",
					"TzT...SnmS",
				},
			},
		},
		/* Useful for commenting out tests. */
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
			test.etc.Node = h.Node()
			command.ExecuteTest(t, test.etc, &command.ExecuteTestOptions{
				RequiresSetup: true,
				SetupContents: test.history,
			})
			command.ChangeTest(t, nil, h)
		})
	}
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

func TestDisjointMatches(t *testing.T) {
	for _, test := range []struct {
		name    string
		matches []*match
		want    []*match
	}{
		{
			name: "handles empty",
		},
		{
			name: "leaves disjoint matches alone",
			matches: []*match{
				{2, 4},
				{14, 18},
				{8, 12},
			},
			want: []*match{
				{2, 4},
				{8, 12},
				{14, 18},
			},
		},
		{
			name: "handles matches that overlap on the same number",
			matches: []*match{
				{2, 4},
				{4, 6},
				{18, 19},
				{12, 18},
			},
			want: []*match{
				{2, 6},
				{12, 19},
			},
		},
		{
			// Indices returned by regex are already of format [start, end)
			name: "leaves matches that are adjacent separate",
			matches: []*match{
				{2, 4},
				{5, 6},
				{19, 20},
				{12, 18},
			},
			want: []*match{
				{2, 4},
				{5, 6},
				{12, 18},
				{19, 20},
			},
		},
		{
			name: "handles overlapping regions",
			matches: []*match{
				{12, 22},
				{5, 6},
				{2, 15},
				{13, 16},
				{20, 25},
				{19, 19},
				{12, 18},
			},
			want: []*match{
				{2, 25},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if diff := cmp.Diff(test.want, disjointMatches(test.matches), cmp.AllowUnexported(match{}), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("disjointMatches(%v) returned diff (-want, +got):\n%s", test.matches, diff)
			}
		})
	}
}
