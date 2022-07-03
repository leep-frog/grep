package grep

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/leep-frog/command"
)

var (
	visitFlag     = command.BoolFlag("cat", 'c', "Run cat command on all files that match")
	filesOnlyFlag = command.BoolFlag("file-only", 'f', "Only check file names")
	dirsOnlyFlag  = command.BoolFlag("dir-only", 'd', "Only check directory names")
)

func FilenameCLI() *Grep {
	return &Grep{
		InputSource: &filename{},
	}
}

type filename struct{}

func (*filename) Name() string    { return "fp" }
func (*filename) Changed() bool   { return false }
func (*filename) Setup() []string { return nil }
func (*filename) Flags() []command.Flag {
	return []command.Flag{
		visitFlag,
		filesOnlyFlag,
		dirsOnlyFlag,
	}
}
func (*filename) MakeNode(n *command.Node) *command.Node { return n }

func (*filename) Process(output command.Output, data *command.Data, f filter) error {
	cat := data.Bool(visitFlag.Name())
	return filepath.WalkDir(startDir, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return output.Stderrf("file not found: %s\n", path)
			}
			return output.Stderrf("failed to access path %q: %v\n", path, err)
		}

		formattedString, ok := apply(f, de.Name(), data)
		if !ok {
			return nil
		}

		if (de.IsDir() && filesOnlyFlag.Get(data)) || (!de.IsDir() && dirsOnlyFlag.Get(data)) {
			return nil
		}

		if cat {
			if !de.IsDir() {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					return output.Stderrf("failed to read file: %v\n", err)
				}
				output.Stdoutln(string(contents))
			}
		} else {
			output.Stdoutln(filepath.Join(filepath.Dir(path), formattedString))
		}
		return nil
	})
}
