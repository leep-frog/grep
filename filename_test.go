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
			d := FilenameCLI()
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

func TestFilename(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       []string
		stubDir    string
		want       *command.ExecuteData
		wantData   *command.Data
		wantStdout []string
		wantStderr []string
		wantErr    error
	}{
		{
			name: "returns all files",
			wantStdout: []string{
				"testing",
				filepath.Join("testing", "lots.txt"),
				filepath.Join("testing", "numbered.txt"),
				filepath.Join("testing", "other"),
				filepath.Join("testing", "other", "other.txt"),
				filepath.Join("testing", "that.py"),
				filepath.Join("testing", "this.txt"),
			},
			wantData: &command.Data{
				Values: map[string]*command.Value{
					patternArgName: command.StringListValue(),
				},
			},
		},
		{
			name:       "errors on walk error",
			stubDir:    "does-not-exist",
			wantStderr: []string{"file not found: does-not-exist"},
			wantErr:    fmt.Errorf("file not found: does-not-exist"),
			wantData: &command.Data{
				Values: map[string]*command.Value{
					patternArgName: command.StringListValue(),
				},
			},
		},
		{
			name: "errors on invalid regex filter",
			args: []string{":)"},
			wantStderr: []string{
				"invalid regex: error parsing regexp: unexpected ): `:)`",
			},
			wantErr: fmt.Errorf("invalid regex: error parsing regexp: unexpected ): `:)`"),
			wantData: &command.Data{
				Values: map[string]*command.Value{
					patternArgName: command.StringListValue(":)"),
				},
			},
		},
		{
			name: "filters out files",
			args: []string{".*.txt"},
			wantStdout: []string{
				filepath.Join("testing", matchColor.Format("lots.txt")),
				filepath.Join("testing", matchColor.Format("numbered.txt")),
				filepath.Join("testing", "other", matchColor.Format("other.txt")),
				filepath.Join("testing", matchColor.Format("this.txt")),
			},
			wantData: &command.Data{
				Values: map[string]*command.Value{
					patternArgName: command.StringListValue(".*.txt"),
				},
			},
		},
		{
			name: "invert filter",
			args: []string{"-v", ".*.go"},
			wantStdout: []string{
				"testing",
				filepath.Join("testing", "lots.txt"),
				filepath.Join("testing", "numbered.txt"),
				filepath.Join("testing", "other"),
				filepath.Join("testing", "other", "other.txt"),
				filepath.Join("testing", "that.py"),
				filepath.Join("testing", "this.txt"),
			},
			wantData: &command.Data{
				Values: map[string]*command.Value{
					patternArgName: command.StringListValue(),
					"invert":       command.StringListValue(".*.go"),
				},
			},
		},
		{
			name: "errors on invalid invert filter",
			args: []string{"-v", ":)"},
			wantStderr: []string{
				"invalid invert regex: error parsing regexp: unexpected ): `:)`",
			},
			wantErr: fmt.Errorf("invalid invert regex: error parsing regexp: unexpected ): `:)`"),
			wantData: &command.Data{
				Values: map[string]*command.Value{
					patternArgName: command.StringListValue(),
					"invert":       command.StringListValue(":)"),
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
			command.ExecuteTest(t, f.Node(), test.args, test.wantErr, test.want, test.wantData, test.wantStdout, test.wantStderr)

			if f.Changed() {
				t.Fatalf("Filename: Execute(%v, %v) marked Changed as true; want false", f, test.args)
			}
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
