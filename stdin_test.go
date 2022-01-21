package grep

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
)

func TestStdinLoad(t *testing.T) {
	for _, test := range []struct {
		name    string
		json    string
		want    *Grep
		WantErr string
	}{
		{
			name: "handles empty string",
			want: &Grep{
				InputSource: &stdin{},
			},
		},
		{
			name:    "handles invalid json",
			json:    "}}",
			WantErr: "failed to unmarshal json for grep object: invalid character",
			want: &Grep{
				InputSource: &stdin{},
			},
		},
		{
			name: "handles valid json",
			json: `{"InputSource": {"Field": "Value"}}`,
			want: &Grep{
				InputSource: &stdin{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := StdinCLI()
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
				cmp.AllowUnexported(Grep{}),
				cmpopts.IgnoreFields(stdin{}, "scanner"),
			}
			if diff := cmp.Diff(test.want, d, opts...); diff != "" {
				t.Errorf("Load(%s) produced diff:\n%s", test.json, diff)
			}
		})
	}
}

func TestStdin(t *testing.T) {
	for _, test := range []struct {
		name  string
		input []string
		etc   *command.ExecuteTestCase
	}{
		{
			name: "works if no stdin",
			etc:  &command.ExecuteTestCase{},
		},
		{
			name: "prints all lines if no args",
			input: []string{
				"alpha",
				"bravo",
				"delta",
			},
			etc: &command.ExecuteTestCase{
				WantStdout: []string{
					"alpha",
					"bravo",
					"delta",
				},
			},
		},
		{
			name: "prints only matching lines",
			input: []string{
				"alpha",
				"bravo",
				"charlie",
				"delta",
			},
			etc: &command.ExecuteTestCase{
				Args: []string{"a$"},
				WantStdout: []string{
					fmt.Sprintf("alph%s", matchColor.Format("a")),
					fmt.Sprintf("delt%s", matchColor.Format("a")),
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						patternArgName: [][]string{{"a$"}},
					},
				},
			},
		},
		{
			name: "works with before flag",
			input: []string{
				"zero",
				"one",
				"two",
				"three",
				"four",
				"five",
				"six",
				"seven",
				"eight",
				"nine",
				"ten",
			},
			etc: &command.ExecuteTestCase{
				Args: []string{"^...$", "-b", "1"},
				WantStdout: []string{
					"zero",
					matchColor.Format("one"),
					matchColor.Format("two"),
					"five",
					matchColor.Format("six"),
					"nine",
					matchColor.Format("ten"),
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						patternArgName:    [][]string{{"^...$"}},
						beforeFlag.Name(): 1,
					},
				},
			},
		},
		{
			name: "works with after flag",
			input: []string{
				"zero",
				"one",
				"two",
				"three",
				"four",
				"five",
				"six",
				"seven",
				"eight",
				"nine",
				"ten",
			},
			etc: &command.ExecuteTestCase{
				Args: []string{"^.....$", "-a", "2"},
				WantStdout: []string{
					matchColor.Format("three"),
					"four",
					"five",
					matchColor.Format("seven"),
					matchColor.Format("eight"),
					"nine",
					"ten",
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						patternArgName:   [][]string{{"^.....$"}},
						afterFlag.Name(): 2,
					},
				},
			},
		},
		{
			name: "works with before and after flags",
			input: []string{
				"zero",
				"one",
				"two",
				"three",
				"four",
				"five",
				"six",
				"seven",
				"eight",
				"nine",
				"ten",
			},
			etc: &command.ExecuteTestCase{
				Args: []string{"five", "-a", "2", "-b", "3"},
				WantStdout: []string{
					"two",
					"three",
					"four",
					matchColor.Format("five"),
					"six",
					"seven",
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						patternArgName:    [][]string{{"five"}},
						afterFlag.Name():  2,
						beforeFlag.Name(): 3,
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			si := &Grep{
				InputSource: &stdin{
					scanner: bufio.NewScanner(strings.NewReader(strings.Join(test.input, "\n"))),
				},
			}
			test.etc.Node = si.Node()
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, nil, si)
		})
	}
}
