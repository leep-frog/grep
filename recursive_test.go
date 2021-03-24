package grep

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/commands/commands"
	"github.com/leep-frog/commands/commandtest"
)

func TestRecursiveLoad(t *testing.T) {
	for _, test := range []struct {
		name    string
		json    string
		want    *Grep
		wantErr string
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
			wantErr: "failed to unmarshal json for recursive grep object: invalid character",
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
			d := RecursiveGrep()
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
				cmp.AllowUnexported(recursive{}),
				cmp.AllowUnexported(Grep{}),
			}
			if diff := cmp.Diff(test.want, d, opts...); diff != "" {
				t.Errorf("Load(%s) produced diff:\n%s", test.json, diff)
			}
		})
	}
}

func TestRecursiveGrep(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		aliases    map[string]string
		stubDir    string
		osOpenErr  error
		want       *commands.WorldState
		wantStdout []string
		wantStderr []string
	}{
		{
			name:    "errors on walk error",
			stubDir: "does-not-exist",
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue(),
				},
			},
			wantStderr: []string{
				`error when walking through file system: failed to access path "does-not-exist": CreateFile does-not-exist: The system cannot find the file specified.`,
			},
		},
		{
			name:      "errors on open error",
			osOpenErr: fmt.Errorf("oops"),
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue(),
				},
			},
			wantStderr: []string{
				fmt.Sprintf(`error when walking through file system: failed to open file %q: oops`, filepath.Join("testing/numbered.txt")),
			},
		},
		{
			name: "finds matches",
			args: []string{"^alpha"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^alpha"),
				},
			},
			wantStdout: []string{
				fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "other", "other.txt")), fmt.Sprintf("%s%s", matchColor.Format("alpha"), " zero")),
				fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "that.py")), matchColor.Format("alpha")),
			},
		},
		{
			name: "file flag filter works",
			args: []string{"^alpha", "-f", ".*.py"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^alpha"),
				},
				Flags: map[string]*commands.Value{
					fileArg.Name(): commands.StringValue(".*.py"),
				},
			},
			wantStdout: []string{
				fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "that.py")), matchColor.Format("alpha")),
			},
		},
		{
			name: "hide file flag works",
			args: []string{"pha[^e]*", "-h"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("pha[^e]*"),
				},
				Flags: map[string]*commands.Value{
					hideFileFlag.Name(): commands.BoolValue(true),
				},
			},
			wantStdout: []string{
				fmt.Sprintf("%s%s%s", "al", matchColor.Format("pha z"), "ero"), // testing/other/other.txt
				fmt.Sprintf("%s%s", "al", matchColor.Format("pha")),            //testing/that.py
			},
		},
		{
			name: "file only flag works",
			args: []string{"^alp", "-l"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^alp"),
				},
				Flags: map[string]*commands.Value{
					fileOnlyFlag.Name(): commands.BoolValue(true),
				},
			},
			wantStdout: []string{
				fileColor.Format(filepath.Join("testing", "other", "other.txt")), // "alpha zero"
				fileColor.Format(filepath.Join("testing", "that.py")),            // "alpha"
			},
		},
		{
			name: "errors on invalid regex in file flag",
			args: []string{"^alpha", "-f", ":)"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^alpha"),
				},
				Flags: map[string]*commands.Value{
					fileArg.Name(): commands.StringValue(":)"),
				},
			},
			wantStderr: []string{
				"invalid filename regex: error parsing regexp: unexpected ): `:)`",
			},
		},
		// -a flag
		{
			name: "returns lines after",
			args: []string{"five", "-a", "3"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("five"),
				},
				Flags: map[string]*commands.Value{
					afterFlag.Name(): commands.IntValue(3),
				},
			},
			wantStdout: []string{
				fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("five")),
				fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "six"),
				fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "seven"),
				fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "eight"),
			},
		},
		{
			name: "returns lines after when file is hidden",
			args: []string{"five", "-h", "-a", "3"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("five"),
				},
				Flags: map[string]*commands.Value{
					afterFlag.Name():    commands.IntValue(3),
					hideFileFlag.Name(): commands.BoolValue(true),
				},
			},
			wantStdout: []string{
				matchColor.Format("five"),
				"six",
				"seven",
				"eight",
			},
		},
		{
			name: "resets after lines if multiple matches",
			args: []string{"^....$", "-f", "numbered.txt", "-a", "2"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^....$"),
				},
				Flags: map[string]*commands.Value{
					afterFlag.Name(): commands.IntValue(2),
					fileArg.Name():   commands.StringValue("numbered.txt"),
				},
			},
			wantStdout: []string{
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
		{
			name: "resets after lines if multiple matches when file is hidden",
			args: []string{"^....$", "-f", "numbered.txt", "-h", "-a", "2"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^....$"),
				},
				Flags: map[string]*commands.Value{
					afterFlag.Name():    commands.IntValue(2),
					fileArg.Name():      commands.StringValue("numbered.txt"),
					hideFileFlag.Name(): commands.BoolValue(true),
				},
			},
			wantStdout: []string{
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
		// -b flag
		{
			name: "returns lines before",
			args: []string{"five", "-b", "3"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("five"),
				},
				Flags: map[string]*commands.Value{
					beforeFlag.Name(): commands.IntValue(3),
				},
			},
			wantStdout: []string{
				fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "two"),
				fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "three"),
				fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), "four"),
				fmt.Sprintf("%s:%s", fileColor.Format(filepath.Join("testing", "numbered.txt")), matchColor.Format("five")),
			},
		},
		{
			name: "returns lines before when file is hidden",
			args: []string{"five", "-h", "-b", "3"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("five"),
				},
				Flags: map[string]*commands.Value{
					beforeFlag.Name():   commands.IntValue(3),
					hideFileFlag.Name(): commands.BoolValue(true),
				},
			},
			wantStdout: []string{
				"two",
				"three",
				"four",
				matchColor.Format("five"),
			},
		},
		{
			name: "returns lines before with overlaps",
			args: []string{"^....$", "-f", "numbered.txt", "-b", "2"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^....$"),
				},
				Flags: map[string]*commands.Value{
					beforeFlag.Name(): commands.IntValue(2),
					fileArg.Name():    commands.StringValue("numbered.txt"),
				},
			},
			wantStdout: []string{
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
		{
			name: "returns lines before with overlaps when file is hidden",
			args: []string{"^....$", "-f", "numbered.txt", "-h", "-b", "2"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^....$"),
				},
				Flags: map[string]*commands.Value{
					beforeFlag.Name():   commands.IntValue(2),
					fileArg.Name():      commands.StringValue("numbered.txt"),
					hideFileFlag.Name(): commands.BoolValue(true),
				},
			},
			wantStdout: []string{
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
		// -a and -b together
		{
			name: "after and before line flags work together",
			args: []string{"^...$", "-f", "numbered.txt", "-a", "2", "-b", "3"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^...$"),
				},
				Flags: map[string]*commands.Value{
					beforeFlag.Name(): commands.IntValue(3),
					afterFlag.Name():  commands.IntValue(2),
					fileArg.Name():    commands.StringValue("numbered.txt"),
				},
			},
			wantStdout: []string{
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
		{
			name: "after and before line flags work together when file is hidden",
			args: []string{"^...$", "-f", "numbered.txt", "-h", "-a", "2", "-b", "3"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("^...$"),
				},
				Flags: map[string]*commands.Value{
					beforeFlag.Name():   commands.IntValue(3),
					afterFlag.Name():    commands.IntValue(2),
					fileArg.Name():      commands.StringValue("numbered.txt"),
					hideFileFlag.Name(): commands.BoolValue(true),
				},
			},
			wantStdout: []string{
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
		// Directory flag (-d).
		{
			name: "fails if unknown directory flag",
			args: []string{"un", "-d", "dev-null"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("un"),
				},
				Flags: map[string]*commands.Value{
					dirFlag.Name(): commands.StringValue("dev-null"),
				},
			},
			wantStderr: []string{
				`unknown alias: "dev-null"`,
			},
		},
		{
			name: "searches in aliased directory instead",
			aliases: map[string]string{
				"ooo": "testing/other",
			},
			args: []string{"alpha", "-d", "ooo"},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue("alpha"),
				},
				Flags: map[string]*commands.Value{
					dirFlag.Name(): commands.StringValue("ooo"),
				},
			},
			wantStdout: []string{
				fmt.Sprintf("%s:%s zero", fileColor.Format(filepath.Join("testing", "other", "other.txt")), matchColor.Format("alpha")),
			},
		},
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
			commandtest.Execute(t, r.Node(), &commands.WorldState{RawArgs: test.args}, test.want, test.wantStdout, test.wantStderr)

			if r.Changed() {
				t.Fatalf("RecursiveGrep: Execute(%v, %v) marked Changed as true; want false", r, test.args)
			}
		})
	}
}

func TestRecusriveMetadata(t *testing.T) {
	c := RecursiveGrep()

	wantName := "recursive-grep"
	if c.Name() != wantName {
		t.Errorf("RecursiveGrep.Name() returned %q; want %q", c.Name(), wantName)
	}

	wantAlias := "rp"
	if c.Alias() != wantAlias {
		t.Errorf("RecursiveGrep.Alias() returned %q; want %q", c.Alias(), wantAlias)
	}

	if c.Option() != nil {
		t.Errorf("RecursiveGrep.Option() returned %v; want nil", c.Option())
	}
}
