package grep

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/leep-frog/cli/cli"
	"github.com/leep-frog/cli/commands"
)

var (
	startDir = "."
	osOpen   = os.Open
)

func RecursiveGrep() cli.CLI {
	return &grep{
		inputSource: &recursive{},
	}
}

type recursive struct{}

func (*recursive) Name() string  { return "recursive-grep" }
func (*recursive) Alias() string { return "rp" }
func (*recursive) Flags() []commands.Flag {
	return []commands.Flag{
		commands.StringFlag("file", 'f', nil),
		// TODO: bool flags:
		// TODO: only match
		// TODO: only filename
	}
}
func (*recursive) Process(args, flags map[string]*commands.Value, ff filterFunc) ([]string, error) {
	var fr *regexp.Regexp
	if fileRegex := flags["file"].String(); fileRegex != nil {
		var err error
		if fr, err = regexp.Compile(*fileRegex); err != nil {
			return nil, fmt.Errorf("invalid filename regex: %v", err)
		}
	}

	var lines []string
	err := filepath.Walk(startDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to access path %q: %v", path, err)
		}

		if fi.IsDir() {
			return nil
		}

		if fr != nil && !fr.MatchString(fi.Name()) {
			return nil
		}

		f, err := osOpen(path)
		if err != nil {
			return fmt.Errorf("failed to open file %q: %v", path, err)
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if ff(scanner.Text()) {
				lines = append(lines, fmt.Sprintf("%s:%s", path, scanner.Text()))
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error when walking through file system: %v", err)
	}
	return lines, nil
}
