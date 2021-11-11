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
				InputSource: &filename{},
			},
		},
		{
			name:    "handles invalid json",
			json:    "}}",
			WantErr: "failed to unmarshal json for grep object: invalid character",
			want: &Grep{
				InputSource: &filename{},
			},
		},
		{
			name: "handles valid json",
			json: `{"Field": "Value"}`,
			want: &Grep{
				InputSource: &filename{},
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
					"validation failed: [ListIsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`",
				},
				WantErr: fmt.Errorf("validation failed: [ListIsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`"),
				WantData: &command.Data{Values: map[string]*command.Value{
					patternArgName: command.StringListValue(":)"),
				}},
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
				WantData: &command.Data{Values: map[string]*command.Value{
					patternArgName: command.StringListValue(".*.txt"),
				}},
			},
		},
		{
			name: "gets files and directories",
			etc: &command.ExecuteTestCase{
				Args: []string{"oth.*"},
				WantStdout: []string{
					filepath.Join("testing", matchColor.Format("other")),
					filepath.Join("testing", "other", matchColor.Format("other.txt")),
				},
				WantData: &command.Data{Values: map[string]*command.Value{
					patternArgName: command.StringListValue("oth.*"),
				}},
			},
		},
		{
			name: "cats files but not directories",
			etc: &command.ExecuteTestCase{
				Args: []string{"oth.*", "-c"},
				WantStdout: []string{
					"alpha zero\necho bravo\n",
				},
				WantData: &command.Data{Values: map[string]*command.Value{
					patternArgName:   command.StringListValue("oth.*"),
					visitFlag.Name(): command.TrueValue(),
				}},
			},
		},
		{
			name: "cats multiple files",
			etc: &command.ExecuteTestCase{
				Args: []string{"^th", "-c"},
				WantStdout: []string{
					"alpha\n",
					"bravo\n",
				},
				WantData: &command.Data{Values: map[string]*command.Value{
					patternArgName:   command.StringListValue("^th"),
					visitFlag.Name(): command.TrueValue(),
				}},
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
				WantData: &command.Data{Values: map[string]*command.Value{
					"invert": command.StringListValue(".*.go"),
				}},
			},
		},
		{
			name: "errors on invalid invert filter",
			etc: &command.ExecuteTestCase{
				Args: []string{"-v", ":)"},
				WantStderr: []string{
					"validation failed: [ListIsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`",
				},
				WantErr: fmt.Errorf("validation failed: [ListIsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`"),
				WantData: &command.Data{Values: map[string]*command.Value{
					"invert": command.StringListValue(":)"),
				}},
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
			command.ExecuteTest(t, test.etc)
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
