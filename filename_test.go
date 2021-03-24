package grep

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/commands/commands"
	"github.com/leep-frog/commands/commandtest"
)

func TestFilenameLoad(t *testing.T) {
	for _, test := range []struct {
		name    string
		json    string
		want    *Grep
		wantErr string
	}{
		{
			name: "handles empty string",
			want: &Grep{
				inputSource: &filename{},
			},
		},
		{
			name:    "handles invalid json",
			json:    "}}",
			wantErr: "failed to unmarshal json for filename grep object: invalid character",
			want: &Grep{
				inputSource: &filename{},
			},
		},
		{
			name: "handles valid json",
			json: `{"Field": "Value"}`,
			want: &Grep{
				inputSource: &filename{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := FilenameGrep()
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
				cmp.AllowUnexported(filename{}),
				cmp.AllowUnexported(Grep{}),
			}
			if diff := cmp.Diff(test.want, d, opts...); diff != "" {
				t.Errorf("Load(%s) produced diff:\n%s", test.json, diff)
			}
		})
	}
}

func TestFilenameGrep(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		stubDir    string
		want       *commands.WorldState
		wantStdout []string
		wantStderr []string
	}{
		{
			name: "returns all files",
			wantStdout: []string{
				"testing",
				filepath.Join("testing", "numbered.txt"),
				filepath.Join("testing", "other"),
				filepath.Join("testing", "other", "other.txt"),
				filepath.Join("testing", "that.py"),
				filepath.Join("testing", "this.txt"),
			},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue(),
				},
			},
		},
		{
			name:    "errors on walk error",
			stubDir: "does-not-exist",
			wantStderr: []string{
				`error when walking through file system: failed to access path "does-not-exist": CreateFile does-not-exist: The system cannot find the file specified.`,
			},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue(),
				},
			},
		},
		{
			name: "errors on invalid regex filter",
			args: []string{":)"},
			wantStderr: []string{
				"invalid regex: error parsing regexp: unexpected ): `:)`",
			},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue(":)"),
				},
			},
		},
		{
			name: "filters out files",
			args: []string{".*.txt"},
			wantStdout: []string{
				filepath.Join("testing", matchColor.Format("numbered.txt")),
				filepath.Join("testing", "other", matchColor.Format("other.txt")),
				filepath.Join("testing", matchColor.Format("this.txt")),
			},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue(".*.txt"),
				},
			},
		},
		{
			name: "invert filter",
			args: []string{"-v", ".*.go"},
			wantStdout: []string{
				"testing",
				filepath.Join("testing", "numbered.txt"),
				filepath.Join("testing", "other"),
				filepath.Join("testing", "other", "other.txt"),
				filepath.Join("testing", "that.py"),
				filepath.Join("testing", "this.txt"),
			},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue(),
				},
				Flags: map[string]*commands.Value{
					"invert": commands.StringListValue(".*.go"),
				},
			},
		},
		{
			name: "errors on invalid invert filter",
			args: []string{"-v", ":)"},
			wantStderr: []string{
				"invalid invert regex: error parsing regexp: unexpected ): `:)`",
			},
			want: &commands.WorldState{
				Args: map[string]*commands.Value{
					patternArgName: commands.StringListValue(),
				},
				Flags: map[string]*commands.Value{
					"invert": commands.StringListValue(":)"),
				},
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

			// Run the test.
			f := FilenameGrep()
			commandtest.Execute(t, f.Node(), &commands.WorldState{RawArgs: test.args}, test.want, test.wantStdout, test.wantStderr)

			if f.Changed() {
				t.Fatalf("FilenameGrep: Execute(%v, %v) marked Changed as true; want false", f, test.args)
			}
		})
	}
}

func TestFilenameMetadata(t *testing.T) {
	c := FilenameGrep()

	wantName := "filename-grep"
	if c.Name() != wantName {
		t.Errorf("FilenameGrep.Name() returned %q; want %q", c.Name(), wantName)
	}

	wantAlias := "fp"
	if c.Alias() != wantAlias {
		t.Errorf("FilenameGrep.Alias() returned %q; want %q", c.Alias(), wantAlias)
	}

	if c.Option() != nil {
		t.Errorf("FilenameGrep.Option() returned %v; want nil", c.Option())
	}
}
