package grep

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
)

var (
	visitFlag     = commander.BoolFlag("cat", 'c', "Run cat command on all files that match")
	filesOnlyFlag = commander.BoolFlag("file-only", 'f', "Only check file names")
	dirsOnlyFlag  = commander.BoolFlag("dir-only", 'd', "Only check directory names")
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
func (*filename) Flags() []commander.FlagInterface {
	return []commander.FlagInterface{
		visitFlag,
		filesOnlyFlag,
		dirsOnlyFlag,
	}
}
func (*filename) MakeNode(n command.Node) command.Node { return n }

func (*filename) Process(output command.Output, data *command.Data, f filter, ss *sliceSet) error {
	cat := data.Bool(visitFlag.Name())
	return filepath.WalkDir(startDir, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return output.Stderrf("file not found: %s\n", path)
			}
			return output.Stderrf("failed to access path %q: %v\n", path, err)
		}

		formattedString, ok := apply(f, de.Name(), data, ss)
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
			dir := filepath.Dir(path)
			if dir != "." {
				output.Stdoutf("%s%c", dir, filepath.Separator)
			}
			applyFormat(output, data, formattedString)
			output.Stdoutln()
		}
		return nil
	})
}
