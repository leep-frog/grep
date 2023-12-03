package grep

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/leep-frog/command/color/colortest"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commandertest"
	"github.com/leep-frog/command/commandtest"
)

func TestStdin(t *testing.T) {
	for _, sc := range []bool{true, false} {
		commandtest.StubValue(t, &defaultColorValue, sc)
		fakeColor := fakeColorFn(sc)
		for _, test := range []struct {
			name  string
			input []string
			etc   *commandtest.ExecuteTestCase
		}{
			{
				name: "works if no stdin",
				etc:  &commandtest.ExecuteTestCase{},
			},
			{
				name: "prints all lines if no args",
				input: []string{
					"alpha",
					"bravo",
					"delta",
				},
				etc: &commandtest.ExecuteTestCase{
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
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"a$"},
					WantStdout: strings.Join([]string{
						fmt.Sprintf("alph%s", fakeColor(matchColor, "a")),
						fmt.Sprintf("delt%s", fakeColor(matchColor, "a")),
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
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"^...$", "-b", "1"},
					WantStdout: strings.Join([]string{
						"zero",
						fakeColor(matchColor, "one"),
						fakeColor(matchColor, "two"),
						"five",
						fakeColor(matchColor, "six"),
						"nine",
						fakeColor(matchColor, "ten"),
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
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"^.....$", "-a", "2"},
					WantStdout: strings.Join([]string{
						fakeColor(matchColor, "three"),
						"four",
						"five",
						fakeColor(matchColor, "seven"),
						fakeColor(matchColor, "eight"),
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
				etc: &commandtest.ExecuteTestCase{
					Args: []string{"five", "-a", "2", "-b", "3"},
					WantStdout: strings.Join([]string{
						"two",
						"three",
						"four",
						fakeColor(matchColor, "five"),
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
			t.Run(testName(sc, test.name), func(t *testing.T) {
				colortest.StubTput(t)
				si := &Grep{
					InputSource: &stdin{
						scanner: bufio.NewScanner(strings.NewReader(strings.Join(test.input, "\n"))),
					},
				}
				test.etc.Node = si.Node()
				commandertest.ExecuteTest(t, test.etc)
				commandertest.ChangeTest(t, nil, si)
			})
		}
	}
}
