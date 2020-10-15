package grep

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/leep-frog/commands/color"
	"github.com/leep-frog/commands/commands"
)

var (
	startDir                                   = "."
	osOpen   func(s string) (io.Reader, error) = func(s string) (io.Reader, error) { return os.Open(s) }

	fileArg      = commands.StringFlag("file", 'f', nil)
	hideFileFlag = commands.BoolFlag("hideFile", 'h')
	fileOnlyFlag = commands.BoolFlag("fileOnly", 'l')
	// TODO: match only flag (-o)

	fileColor = &color.Format{
		Color: color.Cyan,
	}
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
		fileArg,
		hideFileFlag,
		fileOnlyFlag,
	}
}
func (*recursive) Process(cos commands.CommandOS, args, flags map[string]*commands.Value, _ *commands.OptionInfo, ffs filterFuncs) (*commands.ExecutorResponse, bool) {
	hideFile := flags[hideFileFlag.Name()].Bool() != nil && *flags[hideFileFlag.Name()].Bool()
	fmt.Println(hideFile)
	fileOnly := flags[fileOnlyFlag.Name()].Bool() != nil && *flags[fileOnlyFlag.Name()].Bool()

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
			formattedString, ok := ffs.Apply(scanner.Text())
			if !ok {
				continue
			}

			formattedPath := fileColor.Format(path)
			if fileOnly {
				cos.Stdout(formattedPath)
				break
			}

			if hideFile {
				cos.Stdout(formattedString)
			} else {
				cos.Stdout("%s:%s", formattedPath, formattedString)
			}
		}

		return nil
	})
	if err != nil {
		cos.Stderr("error when walking through file system: %v", err)
		return nil, false
	}
	return nil, true
}
