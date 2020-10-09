package grep

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/cli/cli"
	"github.com/leep-frog/cli/commands"
)

func TestRecursiveGrep(t *testing.T) {
	for _, test := range []struct {
		name      string
		args      []string
		stubDir   string
		osOpenErr error
		wantResp  *commands.ExecutorResponse
	}{
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
			name:      "errors on open error",
			osOpenErr: fmt.Errorf("oops"),
			wantResp: &commands.ExecutorResponse{
				Stderr: []string{
					fmt.Sprintf(`error when walking through file system: failed to open file %q: oops`, filepath.Join("testing/other/other.txt")),
				},
			},
		},
		{
			name: "finds matches",
			args: []string{"^alpha"},
			wantResp: &commands.ExecutorResponse{
				Stdout: []string{
					fmt.Sprintf("%s:%s", filepath.Join("testing", "other", "other.txt"), "alpha"),
					fmt.Sprintf("%s:%s", filepath.Join("testing", "that.py"), "alpha"),
				},
			},
		},
		{
			name: "file flag filter works",
			args: []string{"^alpha", "-f", ".*.py"},
			wantResp: &commands.ExecutorResponse{
				Stdout: []string{
					fmt.Sprintf("%s:%s", filepath.Join("testing", "that.py"), "alpha"),
				},
			},
		},
		{
			name: "errors on invalid regex in file flag",
			args: []string{"^alpha", "-f", ":)"},
			wantResp: &commands.ExecutorResponse{
				Stderr: []string{
					"invalid filename regex: error parsing regexp: unexpected ): `:)`",
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

			// Stub os.Open if necessary
			if test.osOpenErr != nil {
				oldOpen := osOpen
				osOpen = func(s string) (*os.File, error) { return nil, test.osOpenErr }
				defer func() { osOpen = oldOpen }()
			}

			c := RecursiveGrep()
			got, err := cli.Execute(c, test.args)
			if err != nil {
				t.Fatalf("RecursiveGrep: Execute(%v, %v) returned error (%v); want nil", c, test.args, err)
			}

			if diff := cmp.Diff(test.wantResp, got); diff != "" {
				t.Fatalf("RecursiveGrep: Execute(%v, %v) produced response diff (-want, +got):\n%s", c, test.args, diff)
			}

			if c.Changed() {
				t.Fatalf("RecursiveGrep: Execute(%v, %v) marked Changed as true; want false", c, test.args)
			}
		})
	}
}

func TestRecusriveMetadata(t *testing.T) {
	c := RecursiveGrep()

	wantName := "recursive-grep"
	if c.Name() != wantName {
		t.Errorf("RecursiveGrep.Name returned %q; want %q", c.Name(), wantName)
	}

	wantAlias := "rp"
	if c.Alias() != wantAlias {
		t.Errorf("RecursiveGrep.Alias returned %q; want %q", c.Alias(), wantAlias)
	}
}
