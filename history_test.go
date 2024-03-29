package grep

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandertest"
	"github.com/leep-frog/command/commandtest"
)

func TestHistory(t *testing.T) {
	for _, sc := range []bool{true, false} {
		commandtest.StubValue(t, &defaultColorValue, sc)
		fakeColor := fakeColorFn(sc)
		fakeInvertedColor := fakeColorFn(!sc)
		for _, test := range []struct {
			name      string
			history   []string
			osOpenErr error
			etc       *commandtest.ExecuteTestCase
		}{
			{
				name: "returns history",
				history: []string{
					"alpha",
					"beta",
					"delta",
				},
				etc: &commandtest.ExecuteTestCase{
					WantStdout: strings.Join([]string{
						"alpha",
						"beta",
						"delta",
						"",
					}, "\n"),
				},
			},
			{
				name: "filters history",
				history: []string{
					"alpha",
					"beta",
					"delta",
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"^.e"},
					WantData: &command.Data{Values: map[string]interface{}{
						patternArgName: [][]string{{"^.e"}},
					}},
					WantStdout: strings.Join([]string{
						fmt.Sprintf("%s%s", fakeColor(matchColor, "be"), "ta"),
						fmt.Sprintf("%s%s", fakeColor(matchColor, "de"), "lta"),
						"",
					}, "\n"),
				},
			},
			{
				name: "filters history with inverted coloring",
				history: []string{
					"alpha",
					"beta",
					"delta",
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"^.e", "-C"},
					WantData: &command.Data{Values: map[string]interface{}{
						patternArgName:   [][]string{{"^.e"}},
						colorFlag.Name(): true,
					}},
					WantStdout: strings.Join([]string{
						fmt.Sprintf("%s%s", fakeInvertedColor(matchColor, "be"), "ta"),
						fmt.Sprintf("%s%s", fakeInvertedColor(matchColor, "de"), "lta"),
						"",
					}, "\n"),
				},
			},
			{
				name: "filters history considering case",
				history: []string{
					"alphA",
					"beta",
					"deltA",
					"zero",
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"^.*A$", "-i"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							caseFlag.Name(): true,
							patternArgName:  [][]string{{"^.*A$"}},
						},
					},
					WantStdout: strings.Join([]string{
						fakeColor(matchColor, "alphA"),
						// fakeColor(matchColor, "beta"),
						fakeColor(matchColor, "deltA"),
						"",
					}, "\n"),
				},
			},
			{
				name:      "errors on os.Open error",
				osOpenErr: fmt.Errorf("darn"),
				etc: &commandtest.ExecuteTestCase{
					WantStderr: "failed to open setup output file: darn\n",
					WantErr:    fmt.Errorf("failed to open setup output file: darn"),
				},
			},
			{
				name: "works with match only",
				history: []string{
					"qwerTyuiop",
					"asdTfghjTkl",
					"TxcvbnmT",
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"T.*T", "-o"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							matchOnlyFlag.Name(): true,
							patternArgName:       [][]string{{"T.*T"}},
						},
					},
					WantStdout: strings.Join([]string{
						"TfghjT",
						"TxcvbnmT",
						"",
					}, "\n"),
				},
			},
			{
				name: "works with match only and overlapping matches",
				history: []string{
					"qwerTyuiop",
					"aSdTfghSjTkl",
					"TxScvbSnmT",
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"T.*T", "S.*S", "-o"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							matchOnlyFlag.Name(): true,
							patternArgName:       [][]string{{"T.*T", "S.*S"}},
						},
					},
					WantStdout: strings.Join([]string{
						"SdTfghSjT",
						"TxScvbSnmT",
						"",
					}, "\n"),
				},
			},
			{
				name: "works with match only and non-overlapping matches",
				history: []string{
					"qwerTyuiop",
					"SaSdfTghjTkl",
					"TzTxcvbSnmS",
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"T.*T", "S.*S", "-o"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							matchOnlyFlag.Name(): true,
							patternArgName:       [][]string{{"T.*T", "S.*S"}},
						},
					},
					WantStdout: strings.Join([]string{
						"SaS...TghjT",
						"TzT...SnmS",
						"",
					}, "\n"),
				},
			},
			{
				name: "matches only whole word",
				history: []string{
					"1 alph",
					"2 aalpha",
					"3 alphaa",
					"4 alp ha",
					"5 alpha",
				},
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"alpha", "-w"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							wholeWordFlag.Name(): true,
							patternArgName:       [][]string{{"alpha"}},
						},
					},
					WantStdout: strings.Join([]string{
						fmt.Sprintf("5 %s", fakeColor(matchColor, "alpha")),
						"",
					}, "\n"),
				},
			},
			/* Useful for commenting out tests. */
		} {
			t.Run(testName(sc, test.name), func(t *testing.T) {
				// Stub os.Open if necessary
				if test.osOpenErr != nil {
					commandtest.StubValue(t, &osOpen, func(s string) (io.Reader, error) { return nil, test.osOpenErr })
				}

				// Run test
				h := HistoryCLI()
				test.etc.Node = h.Node()
				test.etc.RequiresSetup = true
				test.etc.SetupContents = test.history
				commandertest.ExecuteTest(t, test.etc)
				commandertest.ChangeTest(t, nil, h)
			})
		}
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
