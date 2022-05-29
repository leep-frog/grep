package grep

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/leep-frog/command"
)

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
			name: "returns only files",
			etc: &command.ExecuteTestCase{
				Args: []string{"-f"},
				WantStdout: []string{
					filepath.Join("testing", "lots.txt"),
					filepath.Join("testing", "numbered.txt"),
					filepath.Join("testing", "other", "other.txt"),
					filepath.Join("testing", "that.py"),
					filepath.Join("testing", "this.txt"),
				},
				WantData: &command.Data{Values: map[string]interface{}{
					filesOnlyFlag.Name(): true,
				}},
			},
		},
		{
			name: "returns only dirs",
			etc: &command.ExecuteTestCase{
				Args: []string{"-d"},
				WantStdout: []string{
					"testing",
					filepath.Join("testing", "other"),
				},
				WantData: &command.Data{Values: map[string]interface{}{
					dirsOnlyFlag.Name(): true,
				}},
			},
		},
		{
			name: "returns nothing if only files and only dirs",
			etc: &command.ExecuteTestCase{
				Args: []string{"-d", "-f"},
				WantData: &command.Data{Values: map[string]interface{}{
					filesOnlyFlag.Name(): true,
					dirsOnlyFlag.Name():  true,
				}},
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
					"validation failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`",
				},
				WantErr: fmt.Errorf("validation failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`"),
				WantData: &command.Data{Values: map[string]interface{}{
					patternArgName: [][]string{{":)"}},
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
				WantData: &command.Data{Values: map[string]interface{}{
					patternArgName: [][]string{{".*.txt"}},
				}},
			},
		},
		{
			name: "works with OR operator",
			etc: &command.ExecuteTestCase{
				Args: []string{"\\.txt$", ".*s\\.", "|", ".*\\.py$"},
				WantStdout: []string{
					filepath.Join("testing", matchColor.Format("lots.txt")),
					filepath.Join("testing", matchColor.Format("that.py")),
					filepath.Join("testing", matchColor.Format("this.txt")),
				},
				WantData: &command.Data{Values: map[string]interface{}{
					patternArgName: [][]string{
						{"\\.txt$", ".*s\\."},
						{".*\\.py$"},
					},
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
				WantData: &command.Data{Values: map[string]interface{}{
					patternArgName: [][]string{{"oth.*"}},
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
				WantData: &command.Data{
					Values: map[string]interface{}{
						visitFlag.Name(): true,
						patternArgName:   [][]string{{"oth.*"}},
					},
				},
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
				WantData: &command.Data{
					Values: map[string]interface{}{
						visitFlag.Name(): true,
						patternArgName:   [][]string{{"^th"}},
					},
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
				WantData: &command.Data{Values: map[string]interface{}{
					"invert": []string{".*.go"},
				}},
			},
		},
		{
			name: "errors on invalid invert filter",
			etc: &command.ExecuteTestCase{
				Args: []string{"-v", ":)"},
				WantStderr: []string{
					"validation failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`",
				},
				WantErr: fmt.Errorf("validation failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`"),
				WantData: &command.Data{Values: map[string]interface{}{
					"invert": []string{":)"},
				}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Change starting directory
			tmpStart := "testing"
			if test.stubDir != "" {
				tmpStart = test.stubDir
			}
			command.StubValue(t, &startDir, tmpStart)

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
