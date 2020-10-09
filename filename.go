package grep

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/leep-frog/cli/cli"
	"github.com/leep-frog/cli/commands"
)

func FilenameGrep() cli.CLI {
	return &grep{
		inputSource: &filename{},
	}
}

type filename struct{}

func (*filename) Name() string           { return "filename-grep" }
func (*filename) Alias() string          { return "fp" }
func (*filename) Flags() []commands.Flag { return nil }
func (*filename) Process(args, flags map[string]*commands.Value, ff filterFunc) ([]string, error) {
	var filenames []string
	err := filepath.Walk(startDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access path %q: %v", path, err)
		}
		if ff(fi.Name()) {
			filenames = append(filenames, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error when walking through file system: %v", err)
	}
	return filenames, nil
}
