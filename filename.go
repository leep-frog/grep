package grep

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/leep-frog/commands/commands"
)

func FilenameGrep() *Grep {
	return &Grep{
		inputSource: &filename{},
	}
}

type filename struct{}

func (*filename) Name() string                             { return "filename-grep" }
func (*filename) Alias() string                            { return "fp" }
func (*filename) Changed() bool                            { return false }
func (*filename) Option() *commands.Option                 { return nil }
func (*filename) Flags() []commands.Flag                   { return nil }
func (*filename) Subcommands() map[string]commands.Command { return nil }

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

func (*filename) Process(cos commands.CommandOS, args, flags map[string]*commands.Value, _ *commands.OptionInfo, ffs filterFuncs) (*commands.ExecutorResponse, bool) {
	err := filepath.Walk(startDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access path %q: %v", path, err)
		}

		if formattedString, ok := ffs.Apply(fi.Name()); ok {
			cos.Stdout(filepath.Join(filepath.Dir(path), formattedString))
		}
		return nil
	})
	if err != nil {
		cos.Stderr("error when walking through file system: %v", err)
		return nil, false
	}
	return nil, true
}
