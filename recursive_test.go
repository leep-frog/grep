package grep

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
	"github.com/leep-frog/command/color/colortest"
)

func testName(sc bool, name string) string {
	return fmt.Sprintf("[shouldColor=%v] %s", sc, name)
}

func TestRecursive(t *testing.T) {
	for _, sc := range []bool{true, false} {
		command.StubValue(t, &defaultColorValue, sc)
		fakeColor := fakeColorFn(sc)
		fakeColorLine := func(n int) string {
			return fakeColor(lineColor, fmt.Sprintf("%d", n))
		}
		withLine := func(n int, s string) string {
			return fmt.Sprintf("%s:%s", fakeColorLine(n), s)
		}
		withFile := func(s string, fileParts ...string) string {
			return fmt.Sprintf("%s:%s", fakeColor(fileColor, filepath.Join(fileParts...)), s)
		}
		for _, test := range []struct {
			name           string
			aliases        map[string]string
			ignorePatterns map[string]bool
			stubDir        string
			osOpenErr      error
			etc            *command.ExecuteTestCase
			want           *recursive
			dontColor      bool
		}{
			{
				name:    "errors on walk error",
				stubDir: "does-not-exist",
				etc: &command.ExecuteTestCase{
					WantStderr: "file not found: does-not-exist\n",
					WantErr:    fmt.Errorf(`file not found: does-not-exist`),
				},
			},
			{
				name:      "errors on open error",
				osOpenErr: fmt.Errorf("oops"),
				etc: &command.ExecuteTestCase{
					WantStderr: fmt.Sprintf("failed to open file %q: oops\n", filepath.Join("testing", "lots.txt")),
					WantErr:    fmt.Errorf(`failed to open file %q: oops`, filepath.Join("testing", "lots.txt")),
				},
			},
			{
				name: "finds matches",
				etc: &command.ExecuteTestCase{
					Args: []string{"^alpha"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName: [][]string{{"^alpha"}},
						},
					},
					WantStdout: strings.Join([]string{
						withFile(withLine(1, fmt.Sprintf("%s%s", fakeColor(matchColor, "alpha"), " bravo delta")), "testing", "lots.txt"),
						withFile(withLine(3, fmt.Sprintf("%s%s", fakeColor(matchColor, "alpha"), " hello there")), "testing", "lots.txt"),
						withFile(withLine(1, fmt.Sprintf("%s%s", fakeColor(matchColor, "alpha"), " zero")), "testing", "other", "other.txt"),
						withFile(withLine(1, fakeColor(matchColor, "alpha")), "testing", "that.py"),
						"",
					}, "\n"),
				},
			},
			{
				name: "finds matches with percentages",
				etc: &command.ExecuteTestCase{
					Args: []string{"^XYZ.*", "-n", "-h"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							hideFileFlag.Name(): true,
							hideLineFlag.Name(): true,
							patternArgName:      [][]string{{"^XYZ.*"}},
						},
					},
					WantStdout: strings.Join([]string{
						fakeColor(matchColor, "XYZ %s heyo"),
						"",
					}, "\n"),
				},
			},
			{
				name: "file flag filter works",
				etc: &command.ExecuteTestCase{
					Args: []string{"^alpha", "-n", "-f", ".*.py"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"^alpha"}},
							fileArg.Name():      ".*.py",
							hideLineFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						withFile(fakeColor(matchColor, "alpha"), "testing", "that.py"),
						"",
					}, "\n"),
				},
			},
			{
				name: "inverted file flag filter works",
				etc: &command.ExecuteTestCase{
					Args: []string{"^alpha", "-n", "-F", ".*.py"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:       [][]string{{"^alpha"}},
							invertFileArg.Name(): ".*.py",
							hideLineFlag.Name():  true,
						},
					},
					WantStdout: strings.Join([]string{
						withFile(fmt.Sprintf("%s %s", fakeColor(matchColor, "alpha"), "bravo delta"), "testing", "lots.txt"),
						withFile(fmt.Sprintf("%s %s", fakeColor(matchColor, "alpha"), "hello there"), "testing", "lots.txt"),
						withFile(fmt.Sprintf("%s %s", fakeColor(matchColor, "alpha"), "zero"), "testing", "other", "other.txt"),
						"",
					}, "\n"),
				},
			},
			{
				name: "failure if invalid invert file flag",
				etc: &command.ExecuteTestCase{
					Args: []string{"^alpha", "-F", ":)"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:       [][]string{{"^alpha"}},
							invertFileArg.Name(): ":)",
						},
					},
					WantErr:    fmt.Errorf("invalid invert filename regex: error parsing regexp: unexpected ): `:)`"),
					WantStderr: "invalid invert filename regex: error parsing regexp: unexpected ): `:)`\n",
				},
			},
			{
				name: "hide file flag works",
				etc: &command.ExecuteTestCase{
					Args: []string{"pha[^e]*", "-h", "-n"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"pha[^e]*"}},
							hideFileFlag.Name(): true,
							hideLineFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						fmt.Sprintf("%s%s%s", "al", fakeColor(matchColor, "pha bravo d"), "elta"), // testing/lots.txt
						fmt.Sprintf("%s%s", "bravo delta al", fakeColor(matchColor, "pha")),       // testing/lots.txt
						fmt.Sprintf("%s%s%s", "al", fakeColor(matchColor, "pha h"), "ello there"), // testing/lots.txt
						fmt.Sprintf("%s%s%s", "al", fakeColor(matchColor, "pha z"), "ero"),        // testing/other/other.txt
						fmt.Sprintf("%s%s", "al", fakeColor(matchColor, "pha")),                   //testing/that.py
						"",
					}, "\n"),
				},
			},
			{
				name: "colors multiple matches properly",
				etc: &command.ExecuteTestCase{
					Args: []string{"alpha", "bravo", "-h", "-n"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"alpha", "bravo"}},
							hideFileFlag.Name(): true,
							hideLineFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						strings.Join([]string{fakeColor(matchColor, "alpha"), fakeColor(matchColor, "bravo"), "delta"}, " "),
						strings.Join([]string{fakeColor(matchColor, "bravo"), "delta", fakeColor(matchColor, "alpha")}, " "),
						"",
					}, "\n"),
				},
			},
			{
				name: "colors overlapping matches properly",
				etc: &command.ExecuteTestCase{
					Args: []string{"q.*t", "e.*u", "-h"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"q.*t", "e.*u"}},
							hideFileFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						fmt.Sprintf("%s:%s%s", fakeColorLine(7), fakeColor(matchColor, "qwertyu"), "iop"),
						"",
					}, "\n"),
				},
			},
			{
				name: "match only flag works",
				etc: &command.ExecuteTestCase{
					Args: []string{"^alp", "-o"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:       [][]string{{"^alp"}},
							matchOnlyFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						withFile(withLine(1, "alp"), "testing", "lots.txt"),           // "alpha bravo delta"
						withFile(withLine(3, "alp"), "testing", "lots.txt"),           // "alpha bravo delta"
						withFile(withLine(1, "alp"), "testing", "other", "other.txt"), // "alpha zero"
						withFile(withLine(1, "alp"), "testing", "that.py"),            // "alpha"
						"",
					}, "\n"),
				},
			},
			{
				name: "match only flag and no file flag works",
				etc: &command.ExecuteTestCase{
					Args: []string{"^alp", "-o", "-h"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:       [][]string{{"^alp"}},
							matchOnlyFlag.Name(): true,
							hideFileFlag.Name():  true,
						},
					},
					WantStdout: strings.Join([]string{
						withLine(1, "alp"),
						withLine(3, "alp"),
						withLine(1, "alp"),
						withLine(1, "alp"),
						"",
					}, "\n"),
				},
			},
			{
				name: "match only flag works with overlapping",
				etc: &command.ExecuteTestCase{
					Args: []string{"qwerty", "rtyui", "-n", "-o"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:       [][]string{{"qwerty", "rtyui"}},
							matchOnlyFlag.Name(): true,
							hideLineFlag.Name():  true,
						},
					},
					WantStdout: strings.Join([]string{
						withFile("qwertyui", "testing", "lots.txt"),
						"",
					}, "\n"),
				},
			},
			{
				name: "match only flag works with non-overlapping",
				etc: &command.ExecuteTestCase{
					Args: []string{"qw", "op", "ty", "-o"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:       [][]string{{"qw", "op", "ty"}},
							matchOnlyFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						withFile(withLine(7, "qw...ty...op"), "testing", "lots.txt"),
						"",
					}, "\n"),
				},
			},
			{
				name: "file only flag works",
				etc: &command.ExecuteTestCase{
					Args: []string{"^alp", "-l"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"^alp"}},
							fileOnlyFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						fakeColor(fileColor, filepath.Join("testing", "lots.txt")),           // "alpha bravo delta"
						fakeColor(fileColor, filepath.Join("testing", "other", "other.txt")), // "alpha zero"
						fakeColor(fileColor, filepath.Join("testing", "that.py")),            // "alpha"
						"",
					}, "\n"),
				},
			},
			{
				name: "ignore relevant file types",
				ignorePatterns: map[string]bool{
					`\.py$`: true,
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"^alp", "-l"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"^alp"}},
							fileOnlyFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						fakeColor(fileColor, filepath.Join("testing", "lots.txt")),           // "alpha bravo delta"
						fakeColor(fileColor, filepath.Join("testing", "other", "other.txt")), // "alpha zero"
						"",
					}, "\n"),
				},
			},
			{
				name: "ignore ignore patterns",
				ignorePatterns: map[string]bool{
					`\.py$`: true,
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"^alp", "-l", "-x"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:           [][]string{{"^alp"}},
							fileOnlyFlag.Name():      true,
							ignoreIgnoreFiles.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						fakeColor(fileColor, filepath.Join("testing", "lots.txt")),           // "alpha bravo delta"
						fakeColor(fileColor, filepath.Join("testing", "other", "other.txt")), // "alpha zero"
						fakeColor(fileColor, filepath.Join("testing", "that.py")),            // "alpha"
						"",
					}, "\n"),
				},
			},
			{
				name: "errors on invalid regex in file flag",
				etc: &command.ExecuteTestCase{
					Args: []string{"^alpha", "-f", ":)"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName: [][]string{{"^alpha"}},
							fileArg.Name(): ":)",
						},
					},
					WantStderr: "invalid filename regex: error parsing regexp: unexpected ): `:)`\n",
					WantErr:    fmt.Errorf("invalid filename regex: error parsing regexp: unexpected ): `:)`"),
				},
			},
			// -d flag
			{
				name: "returns matches for depth of 2",
				etc: &command.ExecuteTestCase{
					Args: []string{"alpha", "-d", "2"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:   [][]string{{"alpha"}},
							depthFlag.Name(): 2,
						},
					},
					WantStdout: strings.Join([]string{
						withFile(withLine(1, fmt.Sprintf("%s bravo delta", fakeColor(matchColor, "alpha"))), "testing", "lots.txt"),
						withFile(withLine(2, fmt.Sprintf("bravo delta %s", fakeColor(matchColor, "alpha"))), "testing", "lots.txt"),
						withFile(withLine(3, fmt.Sprintf("%s hello there", fakeColor(matchColor, "alpha"))), "testing", "lots.txt"),
						withFile(withLine(1, fmt.Sprintf("%s zero", fakeColor(matchColor, "alpha"))), "testing", "other", "other.txt"),
						withFile(withLine(1, fakeColor(matchColor, "alpha")), "testing", "that.py"),
						"",
					}, "\n"),
				},
			},
			{
				name: "returns all matches for depth of 0",
				etc: &command.ExecuteTestCase{
					Args: []string{"alpha", "-d", "0"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:   [][]string{{"alpha"}},
							depthFlag.Name(): 0,
						},
					},
					WantStdout: strings.Join([]string{
						withFile(withLine(1, fmt.Sprintf("%s bravo delta", fakeColor(matchColor, "alpha"))), "testing", "lots.txt"),
						withFile(withLine(2, fmt.Sprintf("bravo delta %s", fakeColor(matchColor, "alpha"))), "testing", "lots.txt"),
						withFile(withLine(3, fmt.Sprintf("%s hello there", fakeColor(matchColor, "alpha"))), "testing", "lots.txt"),
						withFile(withLine(1, fmt.Sprintf("%s zero", fakeColor(matchColor, "alpha"))), "testing", "other", "other.txt"),
						withFile(withLine(1, fakeColor(matchColor, "alpha")), "testing", "that.py"),
						"",
					}, "\n"),
				},
			},
			{
				name: "returns matches for depth of 1",
				etc: &command.ExecuteTestCase{
					Args: []string{"alpha", "-d", "1"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:   [][]string{{"alpha"}},
							depthFlag.Name(): 1,
						},
					},
					WantStdout: strings.Join([]string{
						withFile(withLine(1, fmt.Sprintf("%s bravo delta", fakeColor(matchColor, "alpha"))), "testing", "lots.txt"),
						withFile(withLine(2, fmt.Sprintf("bravo delta %s", fakeColor(matchColor, "alpha"))), "testing", "lots.txt"),
						withFile(withLine(3, fmt.Sprintf("%s hello there", fakeColor(matchColor, "alpha"))), "testing", "lots.txt"),
						withFile(withLine(1, fakeColor(matchColor, "alpha")), "testing", "that.py"),
						"",
					}, "\n"),
				},
			},
			// -a flag
			{
				name: "returns lines after",
				etc: &command.ExecuteTestCase{
					Args: []string{"five", "-a", "3"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:   [][]string{{"five"}},
							afterFlag.Name(): 3,
						},
					},
					WantStdout: strings.Join([]string{
						withFile(withLine(6, fakeColor(matchColor, "five")), "testing", "numbered.txt"),
						withFile(withLine(7, "six"), "testing", "numbered.txt"),
						withFile(withLine(8, "seven"), "testing", "numbered.txt"),
						withFile(withLine(9, "eight"), "testing", "numbered.txt"),
						"",
					}, "\n"),
				},
			},
			{
				name: "returns lines after when file is hidden",
				etc: &command.ExecuteTestCase{
					Args: []string{"five", "-h", "-a", "3", "-n"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"five"}},
							afterFlag.Name():    3,
							hideFileFlag.Name(): true,
							hideLineFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						fakeColor(matchColor, "five"),
						"six",
						"seven",
						"eight",
						"",
					}, "\n"),
				},
			},
			{
				name: "resets after lines if multiple matches",
				etc: &command.ExecuteTestCase{
					Args: []string{"^....$", "-f", "numbered.txt", "-a", "2", "-n"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"^....$"}},
							afterFlag.Name():    2,
							fileArg.Name():      "numbered.txt",
							hideLineFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						withFile(fakeColor(matchColor, "zero"), "testing", "numbered.txt"),
						withFile("one", "testing", "numbered.txt"),
						withFile("two", "testing", "numbered.txt"),
						withFile(fakeColor(matchColor, "four"), "testing", "numbered.txt"),
						withFile(fakeColor(matchColor, "five"), "testing", "numbered.txt"),
						withFile("six", "testing", "numbered.txt"),
						withFile("seven", "testing", "numbered.txt"),
						withFile(fakeColor(matchColor, "nine"), "testing", "numbered.txt"),
						"",
					}, "\n"),
				},
			},
			{
				name: "resets after lines if multiple matches when file is hidden",
				etc: &command.ExecuteTestCase{
					Args: []string{"^....$", "-f", "numbered.txt", "-h", "-a", "2", "-n"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"^....$"}},
							afterFlag.Name():    2,
							fileArg.Name():      "numbered.txt",
							hideFileFlag.Name(): true,
							hideLineFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						fakeColor(matchColor, "zero"),
						"one",
						"two",
						fakeColor(matchColor, "four"),
						fakeColor(matchColor, "five"),
						"six",
						"seven",
						fakeColor(matchColor, "nine"),
						"",
					}, "\n"),
				},
			},
			// -b flag
			{
				name: "returns lines before",
				etc: &command.ExecuteTestCase{
					Args: []string{"five", "-n", "-b", "3"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"five"}},
							beforeFlag.Name():   3,
							hideLineFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						withFile("two", "testing", "numbered.txt"),
						withFile("three", "testing", "numbered.txt"),
						withFile("four", "testing", "numbered.txt"),
						withFile(fakeColor(matchColor, "five"), "testing", "numbered.txt"),
						"",
					}, "\n"),
				},
			},
			{
				name: "returns lines before when file is hidden",
				etc: &command.ExecuteTestCase{
					Args: []string{"five", "-h", "-b", "3"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"five"}},
							beforeFlag.Name():   3,
							hideFileFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						withLine(3, "two"),
						withLine(4, "three"),
						withLine(5, "four"),
						withLine(6, fakeColor(matchColor, "five")),
						"",
					}, "\n"),
				},
			},
			{
				name: "returns lines before with overlaps",
				etc: &command.ExecuteTestCase{
					Args: []string{"^....$", "-n", "-f", "numbered.txt", "-b", "2"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"^....$"}},
							beforeFlag.Name():   2,
							fileArg.Name():      "numbered.txt",
							hideLineFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						withFile(fakeColor(matchColor, "zero"), "testing", "numbered.txt"),
						withFile("two", "testing", "numbered.txt"),
						withFile("three", "testing", "numbered.txt"),
						withFile(fakeColor(matchColor, "four"), "testing", "numbered.txt"),
						withFile(fakeColor(matchColor, "five"), "testing", "numbered.txt"),
						withFile("seven", "testing", "numbered.txt"),
						withFile("eight", "testing", "numbered.txt"),
						withFile(fakeColor(matchColor, "nine"), "testing", "numbered.txt"),
						"",
					}, "\n"),
				},
			},
			{
				name: "returns lines before with overlaps when file is hidden",
				etc: &command.ExecuteTestCase{
					Args: []string{"^....$", "-f", "numbered.txt", "-h", "-b", "2"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"^....$"}},
							beforeFlag.Name():   2,
							fileArg.Name():      "numbered.txt",
							hideFileFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						withLine(1, fakeColor(matchColor, "zero")),
						withLine(3, "two"),
						withLine(4, "three"),
						withLine(5, fakeColor(matchColor, "four")),
						withLine(6, fakeColor(matchColor, "five")),
						withLine(8, "seven"),
						withLine(9, "eight"),
						withLine(10, fakeColor(matchColor, "nine")),
						"",
					}, "\n"),
				},
			},
			// -a and -b together
			{
				name: "after and before line flags work together",
				etc: &command.ExecuteTestCase{
					Args: []string{"^...$", "-f", "numbered.txt", "-a", "2", "-b", "3"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:    [][]string{{"^...$"}},
							beforeFlag.Name(): 3,
							afterFlag.Name():  2,
							fileArg.Name():    "numbered.txt",
						},
					},
					WantStdout: strings.Join([]string{
						withFile(withLine(1, "zero"), "testing", "numbered.txt"),
						withFile(withLine(2, fakeColor(matchColor, "one")), "testing", "numbered.txt"),
						withFile(withLine(3, fakeColor(matchColor, "two")), "testing", "numbered.txt"),
						withFile(withLine(4, "three"), "testing", "numbered.txt"),
						withFile(withLine(5, "four"), "testing", "numbered.txt"),
						withFile(withLine(6, "five"), "testing", "numbered.txt"),
						withFile(withLine(7, fakeColor(matchColor, "six")), "testing", "numbered.txt"),
						withFile(withLine(8, "seven"), "testing", "numbered.txt"),
						withFile(withLine(9, "eight"), "testing", "numbered.txt"),
						"",
					}, "\n"),
				},
			},
			{
				name: "after and before line flags work together when file is hidden",
				etc: &command.ExecuteTestCase{
					Args: []string{"^...$", "-f", "numbered.txt", "-h", "-a", "2", "-b", "3"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName:      [][]string{{"^...$"}},
							beforeFlag.Name():   3,
							afterFlag.Name():    2,
							fileArg.Name():      "numbered.txt",
							hideFileFlag.Name(): true,
						},
					},
					WantStdout: strings.Join([]string{
						withLine(1, "zero"),
						withLine(2, fakeColor(matchColor, "one")),
						withLine(3, fakeColor(matchColor, "two")),
						withLine(4, "three"),
						withLine(5, "four"),
						withLine(6, "five"),
						withLine(7, fakeColor(matchColor, "six")),
						withLine(8, "seven"),
						withLine(9, "eight"),
						"",
					}, "\n"),
				},
			},
			// Directory flag (-D).
			{
				name: "fails if unknown directory flag",
				etc: &command.ExecuteTestCase{
					Args: []string{"un", "-D", "dev-null"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName: [][]string{{"un"}},
							dirFlag.Name(): "dev-null",
						},
					},
					WantStderr: "unknown alias: \"dev-null\"\n",
					WantErr:    fmt.Errorf(`unknown alias: "dev-null"`),
				},
			},
			{
				name: "searches in aliased directory instead",
				aliases: map[string]string{
					"ooo": "testing/other",
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"alpha", "-D", "ooo"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							patternArgName: [][]string{{"alpha"}},
							dirFlag.Name(): "ooo",
						},
					},
					WantStdout: strings.Join([]string{
						fmt.Sprintf("%s:%s:%s zero", fakeColor(fileColor, filepath.Join("testing", "other", "other.txt")), fakeColorLine(1), fakeColor(matchColor, "alpha")),
						"",
					}, "\n"),
				},
			},
			// Ignore file patterns
			{
				name: "ignore file pattern requires argument",
				etc: &command.ExecuteTestCase{
					Args:       []string{"if"},
					WantStderr: "Branching argument must be one of [a d l]\n",
					WantErr:    fmt.Errorf("Branching argument must be one of [a d l]"),
				},
			},
			{
				name: "ignore file pattern requires valid argument",
				etc: &command.ExecuteTestCase{
					Args:       []string{"if", "uh"},
					WantStderr: "Branching argument must be one of [a d l]\n",
					WantErr:    fmt.Errorf("Branching argument must be one of [a d l]"),
				},
			},
			{
				name: "add ignore file requires valid regex",
				etc: &command.ExecuteTestCase{
					Args: []string{"if", "a", "*"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							ignoreFilePattern.Name(): []string{"*"},
						},
					},
					WantStderr: "validation for \"IGNORE_PATTERN\" failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`\n",
					WantErr:    fmt.Errorf("validation for \"IGNORE_PATTERN\" failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`"),
				},
			},
			{
				name: "add ignore file pattern to empty map",
				etc: &command.ExecuteTestCase{
					Args: []string{"if", "a", ".binary$"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							ignoreFilePattern.Name(): []string{".binary$"},
						},
					},
				},
				want: &recursive{
					IgnoreFilePatterns: map[string]bool{
						".binary$": true,
					},
				},
			},
			{
				name: "adds ignore file pattern to existing map",
				ignorePatterns: map[string]bool{
					"other": true,
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"if", "a", ".binary$"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							ignoreFilePattern.Name(): []string{".binary$"},
						},
					},
				},
				want: &recursive{
					IgnoreFilePatterns: map[string]bool{
						".binary$": true,
						"other":    true,
					},
				},
			},
			{
				name: "adds multiple ignore file pattern to existing map",
				ignorePatterns: map[string]bool{
					"other": true,
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"if", "a", ".bin", "ary$"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							ignoreFilePattern.Name(): []string{".bin", "ary$"},
						},
					},
				},
				want: &recursive{
					IgnoreFilePatterns: map[string]bool{
						".bin":  true,
						"ary$":  true,
						"other": true,
					},
				},
			},
			{
				name: "delete ignore file requires valid regex",
				etc: &command.ExecuteTestCase{
					Args: []string{"if", "d", "*"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							ignoreFilePattern.Name(): []string{"*"},
						},
					},
					WantStderr: "validation for \"IGNORE_PATTERN\" failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`\n",
					WantErr:    fmt.Errorf("validation for \"IGNORE_PATTERN\" failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`"),
				},
			},
			{
				name: "deletes ignore file patterns from empty map",
				etc: &command.ExecuteTestCase{
					Args: []string{"if", "d", ".bin", "ary$"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							ignoreFilePattern.Name(): []string{".bin", "ary$"},
						},
					},
				},
			},
			{
				name: "deletes ignore file patterns map",
				ignorePatterns: map[string]bool{
					".bin":  true,
					"other": true,
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"if", "d", ".bin", "ary$"},
					WantData: &command.Data{
						Values: map[string]interface{}{
							ignoreFilePattern.Name(): []string{".bin", "ary$"},
						},
					},
				},
				want: &recursive{
					IgnoreFilePatterns: map[string]bool{
						"other": true,
					},
				},
			},
			{
				name: "lists ignore file patterns from empty map",
				etc: &command.ExecuteTestCase{
					Args: []string{"if", "l"},
				},
			},
			{
				name: "lists ignore file patterns from empty map",
				ignorePatterns: map[string]bool{
					".bin":  true,
					"other": true,
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"if", "l"},
					WantStdout: strings.Join([]string{
						".bin",
						"other",
						"",
					}, "\n"),
				},
			},
			/* Useful for commenting out tests. */
		} {
			t.Run(testName(sc, test.name), func(t *testing.T) {
				// Change starting directory
				tmpStart := "testing"
				if test.stubDir != "" {
					tmpStart = test.stubDir
				}
				command.StubValue(t, &startDir, tmpStart)
				colortest.StubTput(t)

				// Stub os.Open if necessary
				if test.osOpenErr != nil {
					command.StubValue(t, &osOpen, func(s string) (io.Reader, error) { return nil, test.osOpenErr })
				}

				r := &Grep{
					InputSource: &recursive{
						DirectoryAliases:   test.aliases,
						IgnoreFilePatterns: test.ignorePatterns,
					},
				}
				var g *Grep
				if test.want != nil {
					g = &Grep{
						InputSource: test.want,
					}
				}
				test.etc.Node = r.Node()
				command.ExecuteTest(t, test.etc)
				command.ChangeTest(t, g, r, cmpopts.IgnoreUnexported(recursive{}))
			})
		}
	}
}

func TestAutocomplete(t *testing.T) {
	for _, test := range []struct {
		name string
		r    *recursive
		ctc  *command.CompleteTestCase
	}{
		{
			name: "delete completes ignore file patterns",
			r: &recursive{
				IgnoreFilePatterns: map[string]bool{
					"abc": true,
					"def": true,
					"ghi": true,
				},
			},
			ctc: &command.CompleteTestCase{
				Args: "cmd if d ",
				Want: []string{"abc", "def", "ghi"},
				WantData: &command.Data{
					Values: map[string]interface{}{
						ignoreFilePattern.Name(): []string{""},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := &Grep{test.r}
			test.ctc.Node = g.Node()
			command.CompleteTest(t, test.ctc)
		})
	}
}

func TestRecusriveMetadata(t *testing.T) {
	c := RecursiveCLI()

	wantName := "rp"
	if c.Name() != wantName {
		t.Errorf("Recursive.Name() returned %q; want %q", c.Name(), wantName)
	}

	if c.Setup() != nil {
		t.Errorf("Recursive.Option() returned %v; want nil", c.Setup())
	}
}

func TestUsage(t *testing.T) {
	// Recursive grep
	command.UsageTest(t, &command.UsageTestCase{
		Node: RecursiveCLI().Node(),
		WantString: []string{
			"< { [ PATTERN ... ] | } ... --after|-a --before|-b --case|-i --color|-C --depth|-d --directory|-D --file|-f --file-only|-l --hide-file|-h --hide-lines|-n --ignore-ignore-files|-x --invert|-v --invert-file|-F --match-only|-o --whole-word|-w",
			"",
			"  Commands around global ignore file patterns",
			"  if <",
			"",
			"    Add a global file ignore pattern",
			"    a IGNORE_PATTERN [ IGNORE_PATTERN ... ]",
			"",
			"    Deletes a global file ignore pattern",
			"    d IGNORE_PATTERN [ IGNORE_PATTERN ... ]",
			"",
			"    List global file ignore patterns",
			"    l",
			"",
			"Arguments:",
			"  IGNORE_PATTERN: Files that match these will be ignored",
			"    IsRegex()",
			"  PATTERN: Pattern(s) required to be present in each line. The list breaker acts as an OR operator for groups of regexes",
			"    IsRegex()",
			"",
			"Flags:",
			"  [a] after: Show the matched line and the n lines after it",
			"  [b] before: Show the matched line and the n lines before it",
			"  [i] case: Don't ignore character casing",
			"  [C] color: Force (or unforce) the grep output to include color",
			"  [d] depth: The depth of files to search",
			"  [D] directory: Search through the provided directory instead of pwd",
			"  [f] file: Only select files that match this pattern",
			"  [l] file-only: Only show file names",
			"  [h] hide-file: Don't show file names",
			"  [n] hide-lines: Don't include the line number in the output",
			"  [x] ignore-ignore-files: Ignore the provided IGNORE_PATTERNS",
			"  [v] invert: Pattern(s) required to be absent in each line",
			"  [F] invert-file: Only select files that don't match this pattern",
			"  [o] match-only: Only show the matching segment",
			"  [w] whole-word: Whether or not to search for exact match",
			"",
			"Symbols:",
			command.BranchDesc,
			"  |: List breaker",
		},
	})

	// History grep
	command.UsageTest(t, &command.UsageTestCase{
		Node: HistoryCLI().Node(),
		WantString: []string{
			"{ [ PATTERN ... ] | } ... --case|-i --color|-C --invert|-v --match-only|-o --whole-word|-w",
			"",
			"Arguments:",
			"  PATTERN: Pattern(s) required to be present in each line. The list breaker acts as an OR operator for groups of regexes",
			"    IsRegex()",
			"",
			"Flags:",
			"  [i] case: Don't ignore character casing",
			"  [C] color: Force (or unforce) the grep output to include color",
			"  [v] invert: Pattern(s) required to be absent in each line",
			"  [o] match-only: Only show the matching segment",
			"  [w] whole-word: Whether or not to search for exact match",
			"",
			"Symbols:",
			"  |: List breaker",
		},
	})

	// Filename grep
	command.UsageTest(t, &command.UsageTestCase{
		Node: FilenameCLI().Node(),
		WantString: []string{
			"{ [ PATTERN ... ] | } ... --case|-i --cat|-c --color|-C --dir-only|-d --file-only|-f --invert|-v --match-only|-o --whole-word|-w",
			"",
			"Arguments:",
			"  PATTERN: Pattern(s) required to be present in each line. The list breaker acts as an OR operator for groups of regexes",
			"    IsRegex()",
			"",
			"Flags:",
			"  [i] case: Don't ignore character casing",
			"  [c] cat: Run cat command on all files that match",
			"  [C] color: Force (or unforce) the grep output to include color",
			"  [d] dir-only: Only check directory names",
			"  [f] file-only: Only check file names",
			"  [v] invert: Pattern(s) required to be absent in each line",
			"  [o] match-only: Only show the matching segment",
			"  [w] whole-word: Whether or not to search for exact match",
			"",
			"Symbols:",
			"  |: List breaker",
		},
	})

	// Stdin grep
	command.UsageTest(t, &command.UsageTestCase{
		Node: StdinCLI().Node(),
		WantString: []string{
			"{ [ PATTERN ... ] | } ... --after|-a --before|-b --case|-i --color|-C --invert|-v --match-only|-o --whole-word|-w",
			"",
			"Arguments:",
			"  PATTERN: Pattern(s) required to be present in each line. The list breaker acts as an OR operator for groups of regexes",
			"    IsRegex()",
			"",
			"Flags:",
			"  [a] after: Show the matched line and the n lines after it",
			"  [b] before: Show the matched line and the n lines before it",
			"  [i] case: Don't ignore character casing",
			"  [C] color: Force (or unforce) the grep output to include color",
			"  [v] invert: Pattern(s) required to be absent in each line",
			"  [o] match-only: Only show the matching segment",
			"  [w] whole-word: Whether or not to search for exact match",
			"",
			"Symbols:",
			"  |: List breaker",
		},
	})
}
