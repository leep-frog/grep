package grep

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/leep-frog/command"
)

func TestRecursive(t *testing.T) {
	for _, test := range []struct {
		name           string
		aliases        map[string]string
		ignorePatterns map[string]bool
		stubDir        string
		osOpenErr      error
		etc            *command.ExecuteTestCase
		want           *recursive
	}{
		{
			name:    "errors on walk error",
			stubDir: "does-not-exist",
			etc: &command.ExecuteTestCase{
				WantStderr: []string{`file not found: does-not-exist`},
				WantErr:    fmt.Errorf(`file not found: does-not-exist`),
			},
		},
		{
			name:      "errors on open error",
			osOpenErr: fmt.Errorf("oops"),
			etc: &command.ExecuteTestCase{
				WantStderr: []string{
					fmt.Sprintf(`failed to open file %q: oops`, filepath.Join("testing", "lots.txt")),
				},
				WantErr: fmt.Errorf(`failed to open file %q: oops`, filepath.Join("testing", "lots.txt")),
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
				WantStdout: []string{
					withFile(withLine(1, fmt.Sprintf("%s%s", matchColor.Format("alpha"), " bravo delta")), "testing", "lots.txt"),
					withFile(withLine(3, fmt.Sprintf("%s%s", matchColor.Format("alpha"), " hello there")), "testing", "lots.txt"),
					withFile(withLine(1, fmt.Sprintf("%s%s", matchColor.Format("alpha"), " zero")), "testing", "other", "other.txt"),
					withFile(withLine(1, matchColor.Format("alpha")), "testing", "that.py"),
				},
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
				WantStdout: []string{
					matchColor.Format("XYZ %s heyo"),
				},
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
				WantStdout: []string{
					withFile(matchColor.Format("alpha"), "testing", "that.py"),
				},
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
				WantStdout: []string{
					withFile(fmt.Sprintf("%s %s", matchColor.Format("alpha"), "bravo delta"), "testing", "lots.txt"),
					withFile(fmt.Sprintf("%s %s", matchColor.Format("alpha"), "hello there"), "testing", "lots.txt"),
					withFile(fmt.Sprintf("%s %s", matchColor.Format("alpha"), "zero"), "testing", "other", "other.txt"),
				},
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
				WantStderr: []string{"invalid invert filename regex: error parsing regexp: unexpected ): `:)`"},
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
				WantStdout: []string{
					fmt.Sprintf("%s%s%s", "al", matchColor.Format("pha bravo d"), "elta"), // testing/lots.txt
					fmt.Sprintf("%s%s", "bravo delta al", matchColor.Format("pha")),       // testing/lots.txt
					fmt.Sprintf("%s%s%s", "al", matchColor.Format("pha h"), "ello there"), // testing/lots.txt
					fmt.Sprintf("%s%s%s", "al", matchColor.Format("pha z"), "ero"),        // testing/other/other.txt
					fmt.Sprintf("%s%s", "al", matchColor.Format("pha")),                   //testing/that.py
				},
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
				WantStdout: []string{
					strings.Join([]string{matchColor.Format("alpha"), matchColor.Format("bravo"), "delta"}, " "),
					strings.Join([]string{matchColor.Format("bravo"), "delta", matchColor.Format("alpha")}, " "),
				},
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
				WantStdout: []string{
					fmt.Sprintf("%s:%s%s", colorLine(7), matchColor.Format("qwertyu"), "iop"),
				},
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
				WantStdout: []string{
					withFile(withLine(1, "alp"), "testing", "lots.txt"),           // "alpha bravo delta"
					withFile(withLine(3, "alp"), "testing", "lots.txt"),           // "alpha bravo delta"
					withFile(withLine(1, "alp"), "testing", "other", "other.txt"), // "alpha zero"
					withFile(withLine(1, "alp"), "testing", "that.py"),            // "alpha"
				},
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
				WantStdout: []string{
					withLine(1, "alp"),
					withLine(3, "alp"),
					withLine(1, "alp"),
					withLine(1, "alp"),
				},
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
				WantStdout: []string{
					withFile("qwertyui", "testing", "lots.txt"),
				},
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
				WantStdout: []string{
					withFile(withLine(7, "qw...ty...op"), "testing", "lots.txt"),
				},
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
				WantStdout: []string{
					fileColor.Format(filepath.Join("testing", "lots.txt")),           // "alpha bravo delta"
					fileColor.Format(filepath.Join("testing", "other", "other.txt")), // "alpha zero"
					fileColor.Format(filepath.Join("testing", "that.py")),            // "alpha"
				},
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
				WantStdout: []string{
					fileColor.Format(filepath.Join("testing", "lots.txt")),           // "alpha bravo delta"
					fileColor.Format(filepath.Join("testing", "other", "other.txt")), // "alpha zero"
				},
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
				WantStdout: []string{
					fileColor.Format(filepath.Join("testing", "lots.txt")),           // "alpha bravo delta"
					fileColor.Format(filepath.Join("testing", "other", "other.txt")), // "alpha zero"
					fileColor.Format(filepath.Join("testing", "that.py")),            // "alpha"
				},
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
				WantStderr: []string{
					"invalid filename regex: error parsing regexp: unexpected ): `:)`",
				},
				WantErr: fmt.Errorf("invalid filename regex: error parsing regexp: unexpected ): `:)`"),
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
				WantStdout: []string{
					withFile(withLine(6, matchColor.Format("five")), "testing", "numbered.txt"),
					withFile(withLine(7, "six"), "testing", "numbered.txt"),
					withFile(withLine(8, "seven"), "testing", "numbered.txt"),
					withFile(withLine(9, "eight"), "testing", "numbered.txt"),
				},
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
				WantStdout: []string{
					matchColor.Format("five"),
					"six",
					"seven",
					"eight",
				},
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
				WantStdout: []string{
					withFile(matchColor.Format("zero"), "testing", "numbered.txt"),
					withFile("one", "testing", "numbered.txt"),
					withFile("two", "testing", "numbered.txt"),
					withFile(matchColor.Format("four"), "testing", "numbered.txt"),
					withFile(matchColor.Format("five"), "testing", "numbered.txt"),
					withFile("six", "testing", "numbered.txt"),
					withFile("seven", "testing", "numbered.txt"),
					withFile(matchColor.Format("nine"), "testing", "numbered.txt"),
				},
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
				WantStdout: []string{
					matchColor.Format("zero"),
					"one",
					"two",
					matchColor.Format("four"),
					matchColor.Format("five"),
					"six",
					"seven",
					matchColor.Format("nine"),
				},
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
				WantStdout: []string{
					withFile("two", "testing", "numbered.txt"),
					withFile("three", "testing", "numbered.txt"),
					withFile("four", "testing", "numbered.txt"),
					withFile(matchColor.Format("five"), "testing", "numbered.txt"),
				},
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
				WantStdout: []string{
					withLine(3, "two"),
					withLine(4, "three"),
					withLine(5, "four"),
					withLine(6, matchColor.Format("five")),
				},
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
				WantStdout: []string{
					withFile(matchColor.Format("zero"), "testing", "numbered.txt"),
					withFile("two", "testing", "numbered.txt"),
					withFile("three", "testing", "numbered.txt"),
					withFile(matchColor.Format("four"), "testing", "numbered.txt"),
					withFile(matchColor.Format("five"), "testing", "numbered.txt"),
					withFile("seven", "testing", "numbered.txt"),
					withFile("eight", "testing", "numbered.txt"),
					withFile(matchColor.Format("nine"), "testing", "numbered.txt"),
				},
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
				WantStdout: []string{
					withLine(1, matchColor.Format("zero")),
					withLine(3, "two"),
					withLine(4, "three"),
					withLine(5, matchColor.Format("four")),
					withLine(6, matchColor.Format("five")),
					withLine(8, "seven"),
					withLine(9, "eight"),
					withLine(10, matchColor.Format("nine")),
				},
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
				WantStdout: []string{
					withFile(withLine(1, "zero"), "testing", "numbered.txt"),
					withFile(withLine(2, matchColor.Format("one")), "testing", "numbered.txt"),
					withFile(withLine(3, matchColor.Format("two")), "testing", "numbered.txt"),
					withFile(withLine(4, "three"), "testing", "numbered.txt"),
					withFile(withLine(5, "four"), "testing", "numbered.txt"),
					withFile(withLine(6, "five"), "testing", "numbered.txt"),
					withFile(withLine(7, matchColor.Format("six")), "testing", "numbered.txt"),
					withFile(withLine(8, "seven"), "testing", "numbered.txt"),
					withFile(withLine(9, "eight"), "testing", "numbered.txt"),
				},
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
				WantStdout: []string{
					withLine(1, "zero"),
					withLine(2, matchColor.Format("one")),
					withLine(3, matchColor.Format("two")),
					withLine(4, "three"),
					withLine(5, "four"),
					withLine(6, "five"),
					withLine(7, matchColor.Format("six")),
					withLine(8, "seven"),
					withLine(9, "eight"),
				},
			},
		},
		// Directory flag (-d).
		{
			name: "fails if unknown directory flag",
			etc: &command.ExecuteTestCase{
				Args: []string{"un", "-d", "dev-null"},
				WantData: &command.Data{
					Values: map[string]interface{}{
						patternArgName: [][]string{{"un"}},
						dirFlag.Name(): "dev-null",
					},
				},
				WantStderr: []string{
					`unknown alias: "dev-null"`,
				},
				WantErr: fmt.Errorf(`unknown alias: "dev-null"`),
			},
		},
		{
			name: "searches in aliased directory instead",
			aliases: map[string]string{
				"ooo": "testing/other",
			},
			etc: &command.ExecuteTestCase{
				Args: []string{"alpha", "-d", "ooo"},
				WantData: &command.Data{
					Values: map[string]interface{}{
						patternArgName: [][]string{{"alpha"}},
						dirFlag.Name(): "ooo",
					},
				},
				WantStdout: []string{
					fmt.Sprintf("%s:%s:%s zero", fileColor.Format(filepath.Join("testing", "other", "other.txt")), colorLine(1), matchColor.Format("alpha")),
				},
			},
		},
		// Ignore file patterns
		{
			name: "ignore file pattern requires argument",
			etc: &command.ExecuteTestCase{
				Args:       []string{"if"},
				WantStderr: []string{"Branching argument must be one of [a d l]"},
				WantErr:    fmt.Errorf("Branching argument must be one of [a d l]"),
			},
		},
		{
			name: "ignore file pattern requires valid argument",
			etc: &command.ExecuteTestCase{
				Args:       []string{"if", "uh"},
				WantStderr: []string{"Branching argument must be one of [a d l]"},
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
				WantStderr: []string{"validation failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`"},
				WantErr:    fmt.Errorf("validation failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`"),
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
				WantStderr: []string{"validation failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`"},
				WantErr:    fmt.Errorf("validation failed: [IsRegex] value \"*\" isn't a valid regex: error parsing regexp: missing argument to repetition operator: `*`"),
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
				Args:       []string{"if", "l"},
				WantStdout: []string{".bin", "other"},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			// Change starting directory
			tmpStart := "testing"
			if test.stubDir != "" {
				tmpStart = test.stubDir
			}
			command.StubValue(t, &startDir, tmpStart)

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

func withFile(s string, fileParts ...string) string {
	return fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join(fileParts...)), s)
}

func withLine(n int, s string) string {
	return fmt.Sprintf("%s:%s", colorLine(n), s)
}

func TestUsage(t *testing.T) {
	// Recursive grep
	command.UsageTest(t, &command.UsageTestCase{
		Node: RecursiveCLI().Node(),
		WantString: []string{
			"< { [ PATTERN ... ] | } ... --after|-a --before|-b --case|-i --directory|-d --file|-f --file-only|-l --hide-file|-h --hide-lines|-n --ignore-ignore-files|-x --invert|-v --invert-file|-F --match-only|-o",
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
			"  PATTERN: Pattern(s) required to be present in each line. The list breaker acts as an OR operator for groups of regexes",
			"",
			"Flags:",
			"  [a] after: Show the matched line and the n lines after it",
			"  [b] before: Show the matched line and the n lines before it",
			"  [i] case: Don't ignore character casing",
			"  [d] directory: Search through the provided directory instead of pwd",
			"  [f] file: Only select files that match this pattern",
			"  [l] file-only: Only show file names",
			"  [h] hide-file: Don't show file names",
			"  [n] hide-lines: Don't include the line number in the output",
			"  [x] ignore-ignore-files: Ignore the provided IGNORE_PATTERNS",
			"  [v] invert: Pattern(s) required to be absent in each line",
			"  [F] invert-file: Only select files that don't match this pattern",
			"  [o] match-only: Only show the matching segment",
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
			"{ [ PATTERN ... ] | } ... --case|-i --invert|-v --match-only|-o",
			"",
			"Arguments:",
			"  PATTERN: Pattern(s) required to be present in each line. The list breaker acts as an OR operator for groups of regexes",
			"",
			"Flags:",
			"  [i] case: Don't ignore character casing",
			"  [v] invert: Pattern(s) required to be absent in each line",
			"  [o] match-only: Only show the matching segment",
			"",
			"Symbols:",
			"  |: List breaker",
		},
	})

	// Filename grep
	command.UsageTest(t, &command.UsageTestCase{
		Node: FilenameCLI().Node(),
		WantString: []string{
			"{ [ PATTERN ... ] | } ... --case|-i --cat|-c --dir-only|-d --file-only|-f --invert|-v --match-only|-o",
			"",
			"Arguments:",
			"  PATTERN: Pattern(s) required to be present in each line. The list breaker acts as an OR operator for groups of regexes",
			"",
			"Flags:",
			"  [i] case: Don't ignore character casing",
			"  [c] cat: Run cat command on all files that match",
			"  [d] dir-only: Only check directory names",
			"  [f] file-only: Only check file names",
			"  [v] invert: Pattern(s) required to be absent in each line",
			"  [o] match-only: Only show the matching segment",
			"",
			"Symbols:",
			"  |: List breaker",
		},
	})

	// Stdin grep
	command.UsageTest(t, &command.UsageTestCase{
		Node: StdinCLI().Node(),
		WantString: []string{
			"{ [ PATTERN ... ] | } ... --after|-a --before|-b --case|-i --invert|-v --match-only|-o",
			"",
			"Arguments:",
			"  PATTERN: Pattern(s) required to be present in each line. The list breaker acts as an OR operator for groups of regexes",
			"",
			"Flags:",
			"  [a] after: Show the matched line and the n lines after it",
			"  [b] before: Show the matched line and the n lines before it",
			"  [i] case: Don't ignore character casing",
			"  [v] invert: Pattern(s) required to be absent in each line",
			"  [o] match-only: Only show the matching segment",
			"",
			"Symbols:",
			"  |: List breaker",
		},
	})
}
