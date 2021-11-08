package grep

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/leep-frog/command"
)

var (
	visitFlag = command.BoolFlag("cat-file", 'c', "Run cat command on all files that match")
)

func FilenameCLI() *Grep {
	return &Grep{
		inputSource: &filename{},
	}
}

type filename struct{}

func (*filename) Name() string    { return "fp" }
func (*filename) Changed() bool   { return false }
func (*filename) Setup() []string { return nil }
func (*filename) Flags() []command.Flag {
	return []command.Flag{
		visitFlag,
	}
}
func (*filename) PreProcessors() []command.Processor { return nil }

func (f *filename) Load(jsn string) error {
	if jsn == "" {
		f = &filename{}
		return nil
	}

	if err := json.Unmarshal([]byte(jsn), f); err != nil {
		return fmt.Errorf("failed to unmarshal json for filename grep object: %v", err)
	}
	return nil
}

func (*filename) Process(output command.Output, data *command.Data, ffs filterFuncs) error {
	cat := data.Bool(visitFlag.Name())
	return filepath.Walk(startDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return output.Stderrf("file not found: %s", path)
			}
			return output.Stderrf("failed to access path %q: %v", path, err)
		}

		formattedString, ok := ffs.Apply(fi.Name(), data)

		if !ok {
			return nil
		}

		if cat {
			if !fi.IsDir() {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					return output.Stderrf("failed to read file: %v", err)
				}
				output.Stdout(string(contents))
			}
		} else {
			output.Stdout(filepath.Join(filepath.Dir(path), formattedString))
		}
		return nil
	})
}
