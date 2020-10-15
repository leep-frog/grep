package grep

import (
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

func (*filename) Name() string             { return "filename-grep" }
func (*filename) Alias() string            { return "fp" }
func (*filename) Option() *commands.Option { return nil }
func (*filename) Flags() []commands.Flag   { return nil }
func (*filename) Process(cos commands.CommandOS, args, flags map[string]*commands.Value, _ *commands.OptionInfo, ff filterFunc) (*commands.ExecutorResponse, bool) {
	err := filepath.Walk(startDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access path %q: %v", path, err)
		}
		if ff(fi.Name()) {
			cos.Stdout(path)
		}
		return nil
	})
	if err != nil {
		cos.Stderr("error when walking through file system: %v", err)
		return nil, false
	}
	return &commands.ExecutorResponse{}, true
}
