package grep

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/leep-frog/commands/commands"
)

var (
	startDir                                   = "."
	osOpen   func(s string) (io.Reader, error) = func(s string) (io.Reader, error) { return os.Open(s) }
)

func RecursiveGrep() *Grep {
	return &Grep{
		inputSource: &recursive{},
	}
}

type recursive struct{}

func (*recursive) Name() string             { return "recursive-grep" }
func (*recursive) Alias() string            { return "rp" }
func (*recursive) Option() *commands.Option { return nil }
func (*recursive) Flags() []commands.Flag {
	return []commands.Flag{
		commands.StringFlag("file", 'f', nil),
		// TODO: bool flags:
		// TODO: only match
		// TODO: only filename
	}
}
func (*recursive) Process(cos commands.CommandOS, args, flags map[string]*commands.Value, _ *commands.OptionInfo, ff filterFunc) (*commands.ExecutorResponse, bool) {
	var fr *regexp.Regexp
	if fileRegex := flags["file"].String(); fileRegex != nil {
		var err error
		if fr, err = regexp.Compile(*fileRegex); err != nil {
			cos.Stderr("invalid filename regex: %v", err)
			return nil, false
		}
	}

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
				cos.Stdout("%s:%s", path, scanner.Text())
			}
		}

		return nil
	})
	if err != nil {
		cos.Stderr("error when walking through file system: %v", err)
		return nil, false
	}
	return &commands.ExecutorResponse{}, true
}
