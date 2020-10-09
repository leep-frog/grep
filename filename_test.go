package grep

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/cli/cli"
	"github.com/leep-frog/cli/commands"
)

func TestFilenameGrep(t *testing.T) {
	for _, test := range []struct {
		name     string
		args     []string
		stubDir  string
		wantResp *commands.ExecutorResponse
	}{
		{
			name: "returns all files",
			wantResp: &commands.ExecutorResponse{
				Stdout: []string{
					"testing",
					filepath.Join("testing", "other"),
					filepath.Join("testing", "other", "other.txt"),
					filepath.Join("testing", "that.py"),
					filepath.Join("testing", "this.txt"),
				},
			},
		},
		{
			name:    "errors on walk error",
			stubDir: "does-not-exist",
			wantResp: &commands.ExecutorResponse{
				Stderr: []string{
					`error when walking through file system: failed to access path "does-not-exist": CreateFile does-not-exist: The system cannot find the file specified.`,
				},
			},
		},
		{
			name: "errors on invalid regex filter",
			args: []string{":)"},
			wantResp: &commands.ExecutorResponse{
				Stderr: []string{
					"invalid regex: error parsing regexp: unexpected ): `:)`",
				},
			},
		},
		{
			name: "filters out files",
			args: []string{".*.txt"},
			wantResp: &commands.ExecutorResponse{
				Stdout: []string{
					filepath.Join("testing", "other", "other.txt"),
					filepath.Join("testing", "this.txt"),
				},
			},
		},
		{
			name: "invert filter",
			args: []string{"-v", ".*.go"},
			wantResp: &commands.ExecutorResponse{
				Stdout: []string{
					"testing",
					filepath.Join("testing", "other"),
					filepath.Join("testing", "other", "other.txt"),
					filepath.Join("testing", "that.py"),
					filepath.Join("testing", "this.txt"),
				},
			},
		},
		{
			name: "errors on invalid invert filter",
			args: []string{"-v", ":)"},
			wantResp: &commands.ExecutorResponse{
				Stderr: []string{
					"invalid invert regex: error parsing regexp: unexpected ): `:)`",
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

			// run test
			c := FilenameGrep()
			got, err := cli.Execute(c, test.args)
			if err != nil {
				t.Fatalf("FilenameGrep: Execute(%v, %v) returned error (%v); want nil", c, test.args, err)
			}

			if diff := cmp.Diff(test.wantResp, got); diff != "" {
				t.Fatalf("FilenameGrep: Execute(%v, %v) produced response diff (-want, +got):\n%s", c, test.args, diff)
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
		t.Errorf("FilenameGrep.Name returned %q; want %q", c.Name(), wantName)
	}

	wantAlias := "fp"
	if c.Alias() != wantAlias {
		t.Errorf("FilenameGrep.Alias returned %q; want %q", c.Alias(), wantAlias)
	}
}
