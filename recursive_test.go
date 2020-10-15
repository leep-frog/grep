package grep

import (
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/commands/commands"
)

func TestRecursiveGrep(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		stubDir    string
		osOpenErr  error
		wantOK     bool
		wantResp   *commands.ExecutorResponse
		wantStdout []string
		wantStderr []string
	}{
		{
			name:    "errors on walk error",
			stubDir: "does-not-exist",
			wantStderr: []string{
				`error when walking through file system: failed to access path "does-not-exist": CreateFile does-not-exist: The system cannot find the file specified.`,
			},
		},
		{
			name:      "errors on open error",
			osOpenErr: fmt.Errorf("oops"),
			wantStderr: []string{
				fmt.Sprintf(`error when walking through file system: failed to open file %q: oops`, filepath.Join("testing/other/other.txt")),
			},
		},
		{
			name:     "finds matches",
			args:     []string{"^alpha"},
			wantOK:   true,
			wantResp: &commands.ExecutorResponse{},
			wantStdout: []string{
				fmt.Sprintf("%s:%s", filepath.Join("testing", "other", "other.txt"), "alpha"),
				fmt.Sprintf("%s:%s", filepath.Join("testing", "that.py"), "alpha"),
			},
		},
		{
			name:     "file flag filter works",
			args:     []string{"^alpha", "-f", ".*.py"},
			wantOK:   true,
			wantResp: &commands.ExecutorResponse{},
			wantStdout: []string{
				fmt.Sprintf("%s:%s", filepath.Join("testing", "that.py"), "alpha"),
			},
		},
		{
			name: "errors on invalid regex in file flag",
			args: []string{"^alpha", "-f", ":)"},
			wantStderr: []string{
				"invalid filename regex: error parsing regexp: unexpected ): `:)`",
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

			tcos := &commands.TestCommandOS{}
			c := RecursiveGrep()
			got, ok := commands.Execute(tcos, c.Command(), test.args, nil)
			if ok != test.wantOK {
				t.Fatalf("RecursiveGrep: commands.Execute(%v) returned %v for ok; want %v", test.args, ok, test.wantOK)
			}
			if diff := cmp.Diff(test.wantResp, got); diff != "" {
				t.Fatalf("RecursiveGrep: Execute(%v, %v) produced response diff (-want, +got):\n%s", c, test.args, diff)
			}

			if diff := cmp.Diff(test.wantStdout, tcos.GetStdout()); diff != "" {
				t.Errorf("RecursiveGrep: command.Execute(%v) produced stdout diff (-want, +got):\n%s", test.args, diff)
			}
			if diff := cmp.Diff(test.wantStderr, tcos.GetStderr()); diff != "" {
				t.Errorf("RecursiveGrep: command.Execute(%v) produced stderr diff (-want, +got):\n%s", test.args, diff)
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
