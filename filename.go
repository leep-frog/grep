package grep

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/leep-frog/command"
)

func FilenameCLI() *Grep {
	return &Grep{
		inputSource: &filename{},
	}
}

type filename struct{}

func (*filename) Name() string                       { return "fp" }
func (*filename) Changed() bool                      { return false }
func (*filename) Setup() []string                    { return nil }
func (*filename) Flags() []command.Flag              { return nil }
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
	return filepath.Walk(startDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return output.Stderr("file not found: %s", path)
			}
			return output.Stderr("failed to access path %q: %v", path, err)
		}

		if formattedString, ok := ffs.Apply(fi.Name(), data); ok {
			output.Stdout(filepath.Join(filepath.Dir(path), formattedString))
		}
		return nil
	})
}
