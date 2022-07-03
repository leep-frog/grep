package grep

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/leep-frog/command"
)

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
				WantStdout: strings.Join([]string{
					"alpha",
					"bravo",
					"delta",
					"",
				}, "\n"),
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
				WantStdout: strings.Join([]string{
					fmt.Sprintf("alph%s", matchColor.Format("a")),
					fmt.Sprintf("delt%s", matchColor.Format("a")),
					"",
				}, "\n"),
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
				WantStdout: strings.Join([]string{
					"zero",
					matchColor.Format("one"),
					matchColor.Format("two"),
					"five",
					matchColor.Format("six"),
					"nine",
					matchColor.Format("ten"),
					"",
				}, "\n"),
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
				WantStdout: strings.Join([]string{
					matchColor.Format("three"),
					"four",
					"five",
					matchColor.Format("seven"),
					matchColor.Format("eight"),
					"nine",
					"ten",
					"",
				}, "\n"),
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
				WantStdout: strings.Join([]string{
					"two",
					"three",
					"four",
					matchColor.Format("five"),
					"six",
					"seven",
					"",
				}, "\n"),
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
