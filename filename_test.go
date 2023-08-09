package grep

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/color"
	"github.com/leep-frog/command/color/colortest"
)

// fakeColorFn returns a function that colors the string if sc is set to true.
func fakeColorFn(sc bool) func(f *color.Format, s string) string {
	return func(f *color.Format, s string) string {
		if !sc {
			return s
		}

		preSl := []string(*f)
		prefix := fmt.Sprintf("__tput_%s__", strings.Join(preSl, "_"))

		r := color.Init()
		sufSl := []string(*r)
		suffix := fmt.Sprintf("__tput_%s__", strings.Join(sufSl, "_"))

		return fmt.Sprintf("%s%s%s", prefix, s, suffix)
	}
}

func TestFilename(t *testing.T) {
	for _, sc := range []bool{true, false} {
		command.StubValue(t, &defaultColorValue, sc)
		fakeColor := fakeColorFn(sc)
		for _, test := range []struct {
			name    string
			etc     *command.ExecuteTestCase
			stubDir string
		}{
			{
				name: "returns all files",
				etc: &command.ExecuteTestCase{
					WantStdout: strings.Join([]string{
						"testing",
						filepath.Join("testing", "lots.txt"),
						filepath.Join("testing", "numbered.txt"),
						filepath.Join("testing", "other"),
						filepath.Join("testing", "other", "other.txt"),
						filepath.Join("testing", "that.py"),
						filepath.Join("testing", "this.txt"),
						"",
					}, "\n"),
				},
			},
			{
				name: "returns only files",
				etc: &command.ExecuteTestCase{
					Args: []string{"-f"},
					WantStdout: strings.Join([]string{
						filepath.Join("testing", "lots.txt"),
						filepath.Join("testing", "numbered.txt"),
						filepath.Join("testing", "other", "other.txt"),
						filepath.Join("testing", "that.py"),
						filepath.Join("testing", "this.txt"),
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						filesOnlyFlag.Name(): true,
					}},
				},
			},
			{
				name: "returns only dirs",
				etc: &command.ExecuteTestCase{
					Args: []string{"-d"},
					WantStdout: strings.Join([]string{
						"testing",
						filepath.Join("testing", "other"),
						"",
					}, "\n"),
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
					WantStderr: "file not found: does-not-exist\n",
					WantErr:    fmt.Errorf("file not found: does-not-exist"),
				},
			},
			{
				name: "errors on invalid regex filter",
				etc: &command.ExecuteTestCase{
					Args:       []string{":)"},
					WantStderr: "validation for \"PATTERN\" failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`\n",
					WantErr:    fmt.Errorf("validation for \"PATTERN\" failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`"),
					WantData: &command.Data{Values: map[string]interface{}{
						patternArgName: [][]string{{":)"}},
					}},
				},
			},
			{
				name: "filters out files",
				etc: &command.ExecuteTestCase{
					Args: []string{".*.txt"},
					WantStdout: strings.Join([]string{
						filepath.Join("testing", fakeColor(matchColor, "lots.txt")),
						filepath.Join("testing", fakeColor(matchColor, "numbered.txt")),
						filepath.Join("testing", "other", fakeColor(matchColor, "other.txt")),
						filepath.Join("testing", fakeColor(matchColor, "this.txt")),
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						patternArgName: [][]string{{".*.txt"}},
					}},
				},
			},
			{
				name: "works with OR operator",
				etc: &command.ExecuteTestCase{
					Args: []string{"\\.txt$", ".*s\\.", "|", ".*\\.py$"},
					WantStdout: strings.Join([]string{
						filepath.Join("testing", fakeColor(matchColor, "lots.txt")),
						filepath.Join("testing", fakeColor(matchColor, "that.py")),
						filepath.Join("testing", fakeColor(matchColor, "this.txt")),
						"",
					}, "\n"),
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
					WantStdout: strings.Join([]string{
						filepath.Join("testing", fakeColor(matchColor, "other")),
						filepath.Join("testing", "other", fakeColor(matchColor, "other.txt")),
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						patternArgName: [][]string{{"oth.*"}},
					}},
				},
			},
			{
				name: "cats files but not directories",
				etc: &command.ExecuteTestCase{
					Args: []string{"oth.*", "-c"},
					WantStdout: strings.Join([]string{
						"alpha zero\necho bravo\n",
						"",
					}, "\n"),
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
					WantStdout: strings.Join([]string{
						"alpha\n",
						"bravo\n",
						"",
					}, "\n"),
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
					WantStdout: strings.Join([]string{
						"testing",
						filepath.Join("testing", "lots.txt"),
						filepath.Join("testing", "numbered.txt"),
						filepath.Join("testing", "other"),
						filepath.Join("testing", "other", "other.txt"),
						filepath.Join("testing", "that.py"),
						filepath.Join("testing", "this.txt"),
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						"invert": []string{".*.go"},
					}},
				},
			},
			{
				name: "errors on invalid invert filter",
				etc: &command.ExecuteTestCase{
					Args:       []string{"-v", ":)"},
					WantStderr: "validation for \"invert\" failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`\n",
					WantErr:    fmt.Errorf("validation for \"invert\" failed: [IsRegex] value \":)\" isn't a valid regex: error parsing regexp: unexpected ): `:)`"),
					WantData: &command.Data{Values: map[string]interface{}{
						"invert": []string{":)"},
					}},
				},
			},
			/* Useful for commenting out tests. */
		} {
			t.Run(testName(sc, test.name), func(t *testing.T) {
				// Change starting directory
				tmpStart := "testing"
				if test.stubDir != "" {
					tmpStart = test.stubDir
				}
				command.StubValue(t, &startDir, tmpStart)
				colortest.StubTput(t)

				// Run the test.
				f := FilenameCLI()
				test.etc.Node = f.Node()
				command.ExecuteTest(t, test.etc)
				command.ChangeTest(t, nil, f)
			})
		}
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
