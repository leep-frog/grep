package grep

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/commands/commands"
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
		wantOK     bool
		wantResp   *commands.ExecutorResponse
		wantStdout []string
		wantStderr []string
	}{
		{
			name:   "returns all files",
			wantOK: true,
			wantStdout: []string{
				"testing",
				filepath.Join("testing", "numbered.txt"),
				filepath.Join("testing", "other"),
				filepath.Join("testing", "other", "other.txt"),
				filepath.Join("testing", "that.py"),
				filepath.Join("testing", "this.txt"),
			},
		},
		{
			name:    "errors on walk error",
			stubDir: "does-not-exist",
			wantStderr: []string{
				`error when walking through file system: failed to access path "does-not-exist": CreateFile does-not-exist: The system cannot find the file specified.`,
			},
		},
		{
			name: "errors on invalid regex filter",
			args: []string{":)"},
			wantStderr: []string{
				"invalid regex: error parsing regexp: unexpected ): `:)`",
			},
		},
		{
			name:   "filters out files",
			args:   []string{".*.txt"},
			wantOK: true,
			wantStdout: []string{
				filepath.Join("testing", matchColor.Format("numbered.txt")),
				filepath.Join("testing", "other", matchColor.Format("other.txt")),
				filepath.Join("testing", matchColor.Format("this.txt")),
			},
		},
		{
			name:   "invert filter",
			args:   []string{"-v", ".*.go"},
			wantOK: true,
			wantStdout: []string{
				"testing",
				filepath.Join("testing", "numbered.txt"),
				filepath.Join("testing", "other"),
				filepath.Join("testing", "other", "other.txt"),
				filepath.Join("testing", "that.py"),
				filepath.Join("testing", "this.txt"),
			},
		},
		{
			name: "errors on invalid invert filter",
			args: []string{"-v", ":)"},
			wantStderr: []string{
				"invalid invert regex: error parsing regexp: unexpected ): `:)`",
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

			// run test
			tcos := &commands.TestCommandOS{}
			c := FilenameGrep()
			got, ok := commands.Execute(tcos, c.Command(), test.args, nil)
			if ok != test.wantOK {
				t.Fatalf("FilenameGrep: commands.Execute(%v) returned %v for ok; want %v", test.args, ok, test.wantOK)
			}
			if diff := cmp.Diff(test.wantResp, got); diff != "" {
				t.Fatalf("FilenameGrep: Execute(%v, %v) produced response diff (-want, +got):\n%s", c, test.args, diff)
			}

			if diff := cmp.Diff(test.wantStdout, tcos.GetStdout()); diff != "" {
				t.Errorf("FilenameGrep: command.Execute(%v) produced stdout diff (-want, +got):\n%s", test.args, diff)
			}
			if diff := cmp.Diff(test.wantStderr, tcos.GetStderr()); diff != "" {
				t.Errorf("FilenameGrep: command.Execute(%v) produced stderr diff (-want, +got):\n%s", test.args, diff)
			}

			if c.Changed() {
				t.Fatalf("FilenameGrep: Execute(%v, %v) marked Changed as true; want false", c, test.args)
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
