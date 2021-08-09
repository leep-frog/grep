package grep

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/leep-frog/command"
)

func TestFilenameLoad(t *testing.T) {
	for _, test := range []struct {
		name    string
		json    string
		want    *Grep
		WantErr string
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
			WantErr: "failed to unmarshal json for filename grep object: invalid character",
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
			d := FilenameCLI()
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
				cmp.AllowUnexported(filename{}),
				cmp.AllowUnexported(Grep{}),
			}
			if diff := cmp.Diff(test.want, d, opts...); diff != "" {
				t.Errorf("Load(%s) produced diff:\n%s", test.json, diff)
			}
		})
	}
}

func TestFilename(t *testing.T) {
	for _, test := range []struct {
		name    string
		etc     *command.ExecuteTestCase
		stubDir string
	}{
		{
			name: "returns all files",
			etc: &command.ExecuteTestCase{
				WantStdout: []string{
					"testing",
					filepath.Join("testing", "lots.txt"),
					filepath.Join("testing", "numbered.txt"),
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
			etc: &command.ExecuteTestCase{
				WantStderr: []string{"file not found: does-not-exist"},
				WantErr:    fmt.Errorf("file not found: does-not-exist"),
			},
		},
		{
			name: "errors on invalid regex filter",
			etc: &command.ExecuteTestCase{
				Args: []string{":)"},
				WantStderr: []string{
					"invalid regex: error parsing regexp: unexpected ): `:)`",
				},
				WantErr: fmt.Errorf("invalid regex: error parsing regexp: unexpected ): `:)`"),
				WantData: &command.Data{
					patternArgName: command.StringListValue(":)"),
				},
			},
		},
		{
			name: "filters out files",
			etc: &command.ExecuteTestCase{
				Args: []string{".*.txt"},
				WantStdout: []string{
					filepath.Join("testing", matchColor.Format("lots.txt")),
					filepath.Join("testing", matchColor.Format("numbered.txt")),
					filepath.Join("testing", "other", matchColor.Format("other.txt")),
					filepath.Join("testing", matchColor.Format("this.txt")),
				},
				WantData: &command.Data{
					patternArgName: command.StringListValue(".*.txt"),
				},
			},
		},
		{
			name: "invert filter",
			etc: &command.ExecuteTestCase{
				Args: []string{"-v", ".*.go"},
				WantStdout: []string{
					"testing",
					filepath.Join("testing", "lots.txt"),
					filepath.Join("testing", "numbered.txt"),
					filepath.Join("testing", "other"),
					filepath.Join("testing", "other", "other.txt"),
					filepath.Join("testing", "that.py"),
					filepath.Join("testing", "this.txt"),
				},
				WantData: &command.Data{
					"invert": command.StringListValue(".*.go"),
				},
			},
		},
		{
			name: "errors on invalid invert filter",
			etc: &command.ExecuteTestCase{
				Args: []string{"-v", ":)"},
				WantStderr: []string{
					"invalid invert regex: error parsing regexp: unexpected ): `:)`",
				},
				WantErr: fmt.Errorf("invalid invert regex: error parsing regexp: unexpected ): `:)`"),
				WantData: &command.Data{
					"invert": command.StringListValue(":)"),
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
			f := FilenameCLI()
			test.etc.Node = f.Node()
			command.ExecuteTest(t, test.etc, nil)
			command.ChangeTest(t, nil, f)
		})
	}
}

func TestFilenameMetadata(t *testing.T) {
	c := FilenameCLI()

	wantName := "fp"
	if c.Name() != wantName {
		t.Errorf("Filename.Name() returned %q; want %q", c.Name(), wantName)
	}

	if c.Setup() != nil {
		t.Errorf("Filename.Setup() returned %v; want nil", c.Setup())
	}
}
