package grep

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

func TestRecursiveLoad(t *testing.T) {
	for _, test := range []struct {
		name    string
		json    string
		want    *Grep
		WantErr string
	}{
		{
			name: "handles empty string",
			want: &Grep{
				inputSource: &recursive{},
			},
		},
		{
			name:    "handles invalid json",
			json:    "}}",
			WantErr: "failed to unmarshal json for recursive grep object: invalid character",
			want: &Grep{
				inputSource: &recursive{},
			},
		},
		{
			name: "handles valid json",
			json: `{"Field": "Value"}`,
			want: &Grep{
				inputSource: &recursive{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := RecursiveCLI()
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
				cmp.AllowUnexported(recursive{}),
				cmp.AllowUnexported(Grep{}),
			}
			if diff := cmp.Diff(test.want, d, opts...); diff != "" {
				t.Errorf("Load(%s) produced diff:\n%s", test.json, diff)
			}
		})
	}
}

func TestRecursive(t *testing.T) {
	for _, test := range []struct {
		name      string
		aliases   map[string]string
		stubDir   string
		osOpenErr error
		etc       *command.ExecuteTestCase
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
					patternArgName: command.StringListValue("^alpha"),
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
					patternArgName:      command.StringListValue("^XYZ.*"),
					hideFileFlag.Name(): command.TrueValue(),
					hideLineFlag.Name(): command.TrueValue(),
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
					patternArgName:      command.StringListValue("^alpha"),
					fileArg.Name():      command.StringValue(".*.py"),
					hideLineFlag.Name(): command.TrueValue(),
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
					patternArgName:       command.StringListValue("^alpha"),
					invertFileArg.Name(): command.StringValue(".*.py"),
					hideLineFlag.Name():  command.TrueValue(),
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
					patternArgName:       command.StringListValue("^alpha"),
					invertFileArg.Name(): command.StringValue(":)"),
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
					patternArgName:      command.StringListValue("pha[^e]*"),
					hideFileFlag.Name(): command.TrueValue(),
					hideLineFlag.Name(): command.TrueValue(),
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
					patternArgName:      command.StringListValue("alpha", "bravo"),
					hideFileFlag.Name(): command.TrueValue(),
					hideLineFlag.Name(): command.TrueValue(),
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
					patternArgName:      command.StringListValue("q.*t", "e.*u"),
					hideFileFlag.Name(): command.TrueValue(),
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
					patternArgName:       command.StringListValue("^alp"),
					matchOnlyFlag.Name(): command.TrueValue(),
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
					patternArgName:       command.StringListValue("^alp"),
					matchOnlyFlag.Name(): command.TrueValue(),
					hideFileFlag.Name():  command.TrueValue(),
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
					patternArgName:       command.StringListValue("qwerty", "rtyui"),
					matchOnlyFlag.Name(): command.TrueValue(),
					hideLineFlag.Name():  command.TrueValue(),
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
					patternArgName:       command.StringListValue("qw", "op", "ty"),
					matchOnlyFlag.Name(): command.TrueValue(),
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
					patternArgName:      command.StringListValue("^alp"),
					fileOnlyFlag.Name(): command.TrueValue(),
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
					patternArgName: command.StringListValue("^alpha"),
					fileArg.Name(): command.StringValue(":)"),
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
					patternArgName:   command.StringListValue("five"),
					afterFlag.Name(): command.IntValue(3),
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
					patternArgName:      command.StringListValue("five"),
					afterFlag.Name():    command.IntValue(3),
					hideFileFlag.Name(): command.TrueValue(),
					hideLineFlag.Name(): command.TrueValue(),
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
					patternArgName:      command.StringListValue("^....$"),
					afterFlag.Name():    command.IntValue(2),
					fileArg.Name():      command.StringValue("numbered.txt"),
					hideLineFlag.Name(): command.TrueValue(),
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
					patternArgName:      command.StringListValue("^....$"),
					afterFlag.Name():    command.IntValue(2),
					fileArg.Name():      command.StringValue("numbered.txt"),
					hideFileFlag.Name(): command.TrueValue(),
					hideLineFlag.Name(): command.TrueValue(),
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
					patternArgName:      command.StringListValue("five"),
					beforeFlag.Name():   command.IntValue(3),
					hideLineFlag.Name(): command.TrueValue(),
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
					patternArgName:      command.StringListValue("five"),
					beforeFlag.Name():   command.IntValue(3),
					hideFileFlag.Name(): command.TrueValue(),
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
					patternArgName:      command.StringListValue("^....$"),
					beforeFlag.Name():   command.IntValue(2),
					fileArg.Name():      command.StringValue("numbered.txt"),
					hideLineFlag.Name(): command.TrueValue(),
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
					patternArgName:      command.StringListValue("^....$"),
					beforeFlag.Name():   command.IntValue(2),
					fileArg.Name():      command.StringValue("numbered.txt"),
					hideFileFlag.Name(): command.TrueValue(),
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
					patternArgName:    command.StringListValue("^...$"),
					beforeFlag.Name(): command.IntValue(3),
					afterFlag.Name():  command.IntValue(2),
					fileArg.Name():    command.StringValue("numbered.txt"),
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
					patternArgName:      command.StringListValue("^...$"),
					beforeFlag.Name():   command.IntValue(3),
					afterFlag.Name():    command.IntValue(2),
					fileArg.Name():      command.StringValue("numbered.txt"),
					hideFileFlag.Name(): command.TrueValue(),
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
					patternArgName: command.StringListValue("un"),
					dirFlag.Name(): command.StringValue("dev-null"),
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
					patternArgName: command.StringListValue("alpha"),
					dirFlag.Name(): command.StringValue("ooo"),
				},
				WantStdout: []string{
					fmt.Sprintf("%s:%s:%s zero", fileColor.Format(filepath.Join("testing", "other", "other.txt")), colorLine(1), matchColor.Format("alpha")),
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			// Change starting directory
			oldStart := startDir
			if test.stubDir == "" {
				startDir = "testing"
			} else {
				startDir = test.stubDir
			}
			defer func() { startDir = oldStart }()

			// Stub os.Open if necessary
			if test.osOpenErr != nil {
				oldOpen := osOpen
				osOpen = func(s string) (io.Reader, error) { return nil, test.osOpenErr }
				defer func() { osOpen = oldOpen }()
			}

			r := &Grep{
				inputSource: &recursive{
					DirectoryAliases: test.aliases,
				},
			}
			test.etc.Node = r.Node()
			command.ExecuteTest(t, test.etc)
			command.ChangeTest(t, nil, r)
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
			"[ PATTERN ... ] --after|-a --before|-b --directory|-d --file|-f --file-only|-l --hide-file|-h --hide-lines|-n --ignore-case|-i --invert|-v --invert-file|-F --match-only|-o",
			"",
			"Arguments:",
			"  PATTERN: Pattern(s) required to be present in each line",
			"",
			"Flags:",
			"  after: Show the matched line and the n lines after it",
			"  before: Show the matched line and the n lines before it",
			"  directory: Search through the provided directory instead of pwd",
			"  file: Only select files that match this pattern",
			"  file-only: Only show file names",
			"  hide-file: Don't show file names",
			"  hide-lines: Don't include the line number in the output",
			"  ignore-case: Ignore character casing",
			"  invert: Pattern(s) required to be absent in each line",
			"  invert-file: Only select files that don't match this pattern",
			"  match-only: Only show the matching segment",
		},
	})

	// History grep
	command.UsageTest(t, &command.UsageTestCase{
		Node: HistoryCLI().Node(),
		WantString: []string{
			"SETUP_FILE [ PATTERN ... ] --ignore-case|-i --invert|-v --match-only|-o",
			"",
			"Arguments:",
			"  PATTERN: Pattern(s) required to be present in each line",
			"  SETUP_FILE: file used to run setup for command",
			"",
			"Flags:",
			"  ignore-case: Ignore character casing",
			"  invert: Pattern(s) required to be absent in each line",
			"  match-only: Only show the matching segment",
		},
	})

	// Filename grep
	command.UsageTest(t, &command.UsageTestCase{
		Node: FilenameCLI().Node(),
		WantString: []string{
			"[ PATTERN ... ] --ignore-case|-i --invert|-v --match-only|-o",
			"",
			"Arguments:",
			"  PATTERN: Pattern(s) required to be present in each line",
			"",
			"Flags:",
			"  ignore-case: Ignore character casing",
			"  invert: Pattern(s) required to be absent in each line",
			"  match-only: Only show the matching segment",
		},
	})

	// Stdin grep
	command.UsageTest(t, &command.UsageTestCase{
		Node: StdinCLI().Node(),
		WantString: []string{
			"[ PATTERN ... ] --after|-a --before|-b --ignore-case|-i --invert|-v --match-only|-o",
			"",
			"Arguments:",
			"  PATTERN: Pattern(s) required to be present in each line",
			"",
			"Flags:",
			"  after: Show the matched line and the n lines after it",
			"  before: Show the matched line and the n lines before it",
			"  ignore-case: Ignore character casing",
			"  invert: Pattern(s) required to be absent in each line",
			"  match-only: Only show the matching segment",
		},
	})
}
