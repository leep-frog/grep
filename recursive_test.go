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
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "lots.txt")), fmt.Sprintf("%s%s", matchColor.Format("alpha"), " bravo delta")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "lots.txt")), fmt.Sprintf("%s%s", matchColor.Format("alpha"), " hello there")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "other", "other.txt")), fmt.Sprintf("%s%s", matchColor.Format("alpha"), " zero")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "that.py")), matchColor.Format("alpha")),
				},
			},
		},
		{
			name: "file flag filter works",
			etc: &command.ExecuteTestCase{
				Args: []string{"^alpha", "-f", ".*.py"},
				WantData: &command.Data{
					patternArgName: command.StringListValue("^alpha"),
					fileArg.Name(): command.StringValue(".*.py"),
				},
				WantStdout: []string{
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "that.py")), matchColor.Format("alpha")),
				},
			},
		},
		{
			name: "inverted file flag filter works",
			etc: &command.ExecuteTestCase{
				Args: []string{"^alpha", "-F", ".*.py"},
				WantData: &command.Data{
					patternArgName:       command.StringListValue("^alpha"),
					invertFileArg.Name(): command.StringValue(".*.py"),
				},
				WantStdout: []string{
					fmt.Sprintf("%s:%s %s", fileColor.Format(filepath.Join("testing", "lots.txt")), matchColor.Format("alpha"), "bravo delta"),
					fmt.Sprintf("%s:%s %s", fileColor.Format(filepath.Join("testing", "lots.txt")), matchColor.Format("alpha"), "hello there"),
					fmt.Sprintf("%s:%s %s", fileColor.Format(filepath.Join("testing", "other", "other.txt")), matchColor.Format("alpha"), "zero"),
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
				Args: []string{"pha[^e]*", "-h"},
				WantData: &command.Data{
					patternArgName:      command.StringListValue("pha[^e]*"),
					hideFileFlag.Name(): command.BoolValue(true),
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
				Args: []string{"alpha", "bravo", "-h"},
				WantData: &command.Data{
					patternArgName:      command.StringListValue("alpha", "bravo"),
					hideFileFlag.Name(): command.BoolValue(true),
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
					hideFileFlag.Name(): command.BoolValue(true),
				},
				WantStdout: []string{
					fmt.Sprintf("%s%s", matchColor.Format("qwertyu"), "iop"),
				},
			},
		},
		{
			name: "match only flag works",
			etc: &command.ExecuteTestCase{
				Args: []string{"^alp", "-o"},
				WantData: &command.Data{
					patternArgName:       command.StringListValue("^alp"),
					matchOnlyFlag.Name(): command.BoolValue(true),
				},
				WantStdout: []string{
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "lots.txt")), "alp"),           // "alpha bravo delta"
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "lots.txt")), "alp"),           // "alpha bravo delta"
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "other", "other.txt")), "alp"), // "alpha zero"
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "that.py")), "alp"),            // "alpha"
				},
			},
		},
		{
			name: "match only flag and no file flag works",
			etc: &command.ExecuteTestCase{
				Args: []string{"^alp", "-o", "-h"},
				WantData: &command.Data{
					patternArgName:       command.StringListValue("^alp"),
					matchOnlyFlag.Name(): command.BoolValue(true),
					hideFileFlag.Name():  command.BoolValue(true),
				},
				WantStdout: []string{
					"alp",
					"alp",
					"alp",
					"alp",
				},
			},
		},
		{
			name: "match only flag works with overlapping",
			etc: &command.ExecuteTestCase{
				Args: []string{"qwerty", "rtyui", "-o"},
				WantData: &command.Data{
					patternArgName:       command.StringListValue("qwerty", "rtyui"),
					matchOnlyFlag.Name(): command.BoolValue(true),
				},
				WantStdout: []string{
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "lots.txt")), "qwertyui"),
				},
			},
		},
		{
			name: "match only flag works with non-overlapping",
			etc: &command.ExecuteTestCase{
				Args: []string{"qw", "op", "ty", "-o"},
				WantData: &command.Data{
					patternArgName:       command.StringListValue("qw", "op", "ty"),
					matchOnlyFlag.Name(): command.BoolValue(true),
				},
				WantStdout: []string{
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "lots.txt")), "qw...ty...op"),
				},
			},
		},
		{
			name: "file only flag works",
			etc: &command.ExecuteTestCase{
				Args: []string{"^alp", "-l"},
				WantData: &command.Data{
					patternArgName:      command.StringListValue("^alp"),
					fileOnlyFlag.Name(): command.BoolValue(true),
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
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("five")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "six"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "seven"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "eight"),
				},
			},
		},
		{
			name: "returns lines after when file is hidden",
			etc: &command.ExecuteTestCase{
				Args: []string{"five", "-h", "-a", "3"},
				WantData: &command.Data{
					patternArgName:      command.StringListValue("five"),
					afterFlag.Name():    command.IntValue(3),
					hideFileFlag.Name(): command.BoolValue(true),
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
				Args: []string{"^....$", "-f", "numbered.txt", "-a", "2"},
				WantData: &command.Data{
					patternArgName:   command.StringListValue("^....$"),
					afterFlag.Name(): command.IntValue(2),
					fileArg.Name():   command.StringValue("numbered.txt"),
				},
				WantStdout: []string{
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("zero")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "one"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "two"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("four")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("five")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "six"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "seven"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("nine")),
				},
			},
		},
		{
			name: "resets after lines if multiple matches when file is hidden",
			etc: &command.ExecuteTestCase{
				Args: []string{"^....$", "-f", "numbered.txt", "-h", "-a", "2"},
				WantData: &command.Data{
					patternArgName:      command.StringListValue("^....$"),
					afterFlag.Name():    command.IntValue(2),
					fileArg.Name():      command.StringValue("numbered.txt"),
					hideFileFlag.Name(): command.BoolValue(true),
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
				Args: []string{"five", "-b", "3"},
				WantData: &command.Data{
					patternArgName:    command.StringListValue("five"),
					beforeFlag.Name(): command.IntValue(3),
				},
				WantStdout: []string{
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "two"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "three"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "four"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("five")),
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
					hideFileFlag.Name(): command.BoolValue(true),
				},
				WantStdout: []string{
					"two",
					"three",
					"four",
					matchColor.Format("five"),
				},
			},
		},
		{
			name: "returns lines before with overlaps",
			etc: &command.ExecuteTestCase{
				Args: []string{"^....$", "-f", "numbered.txt", "-b", "2"},
				WantData: &command.Data{
					patternArgName:    command.StringListValue("^....$"),
					beforeFlag.Name(): command.IntValue(2),
					fileArg.Name():    command.StringValue("numbered.txt"),
				},
				WantStdout: []string{
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("zero")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "two"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "three"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("four")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("five")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "seven"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "eight"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("nine")),
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
					hideFileFlag.Name(): command.BoolValue(true),
				},
				WantStdout: []string{
					matchColor.Format("zero"),
					"two",
					"three",
					matchColor.Format("four"),
					matchColor.Format("five"),
					"seven",
					"eight",
					matchColor.Format("nine"),
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
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "zero"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("one")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("two")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "three"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "four"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "five"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("six")),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "seven"),
					fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "eight"),
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
					hideFileFlag.Name(): command.BoolValue(true),
				},
				WantStdout: []string{
					"zero",
					matchColor.Format("one"),
					matchColor.Format("two"),
					"three",
					"four",
					"five",
					matchColor.Format("six"),
					"seven",
					"eight",
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
					fmt.Sprintf("%s:%s zero", fileColor.Format(filepath.Join("testing", "other", "other.txt")), matchColor.Format("alpha")),
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
			command.ExecuteTest(t, test.etc, nil)
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
