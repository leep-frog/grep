package grep

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/color"
)

var (
	startDir                                   = "."
	osOpen   func(s string) (io.Reader, error) = func(s string) (io.Reader, error) { return os.Open(s) }

	// Only select files that match pattern.
	fileArg = command.StringFlag("file", 'f')
	// Only select files that match pattern.
	invertFileArg = command.StringFlag("invertFile", 'F')
	// Don't show file names
	hideFileFlag = command.BoolFlag("hideFile", 'h')
	// Only show file names (hide lines).
	fileOnlyFlag = command.BoolFlag("fileOnly", 'l')
	// Show the matched line and the `n` lines before it.
	beforeFlag = command.IntFlag("before", 'b')
	// Show the matched line and the `n` lines after it.
	afterFlag = command.IntFlag("after", 'a')
	// Directory flag to search through an aliased directory instead of pwd.
	dirFlag = command.StringFlag("directory", 'd', &command.Completor{
		SuggestionFetcher: &command.FileFetcher{
			IgnoreFiles: true,
		},
	})

	fileColor = &color.Format{
		Color: color.Cyan,
	}
)

func RecursiveCLI() *Grep {
	return &Grep{
		inputSource: &recursive{},
	}
}

type recursive struct {
	DirectoryAliases map[string]string
	changed          bool
}

func (*recursive) Name() string    { return "rp" }
func (*recursive) Setup() []string { return nil }
func (*recursive) Flags() []command.Flag {
	return []command.Flag{
		fileArg,
		invertFileArg,
		hideFileFlag,
		fileOnlyFlag,
		beforeFlag,
		afterFlag,
		dirFlag,
	}
}
func (*recursive) PreProcessors() []command.Processor { return nil }

func (r *recursive) Load(jsn string) error {
	if jsn == "" {
		r = &recursive{}
		return nil
	}

	if err := json.Unmarshal([]byte(jsn), r); err != nil {
		return fmt.Errorf("failed to unmarshal json for recursive grep object: %v", err)
	}
	return nil
}

func (r *recursive) Changed() bool {
	return r.changed
}

func (r *recursive) Process(output command.Output, data *command.Data, ffs filterFuncs) error {
	linesBefore := data.Int(beforeFlag.Name())
	linesAfter := data.Int(afterFlag.Name())

	var fr *regexp.Regexp

	if data.HasArg(fileArg.Name()) {
		f := data.String(fileArg.Name())
		var err error
		if fr, err = regexp.Compile(f); err != nil {
			return output.Stderr("invalid filename regex: %v", err)
		}
	}

	var ifr *regexp.Regexp
	if data.HasArg(invertFileArg.Name()) {
		f := data.String(invertFileArg.Name())
		var err error
		if ifr, err = regexp.Compile(f); err != nil {
			return output.Stderr("invalid invert filename regex: %v", err)
		}
	}

	dir := startDir
	if data.HasArg(dirFlag.Name()) {
		da := data.String(dirFlag.Name())
		var ok bool
		dir, ok = r.DirectoryAliases[da]
		if !ok {
			return output.Stderr("unknown alias: %q", da)
		}
	}

	return filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return output.Stderr("file not found: %s", path)
			}
			return output.Stderr("failed to access path %q: %v", path, err)
		}

		if fi.IsDir() {
			return nil
		}

		if fr != nil && !fr.MatchString(fi.Name()) {
			return nil
		}

		if ifr != nil && ifr.MatchString(fi.Name()) {
			return nil
		}

		f, err := osOpen(path)
		if err != nil {
			return output.Stderr("failed to open file %q: %v", path, err)
		}

		list := &linkedList{}
		linesSinceMatch := linesAfter
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			formattedString, ok := ffs.Apply(scanner.Text(), data)
			if !ok {
				linesSinceMatch++
				if linesSinceMatch > linesAfter {
					list.pushBack(scanner.Text())
					if list.length > linesBefore {
						list.pop()
					}
					continue
				}
				formattedString = scanner.Text()
			} else {
				linesSinceMatch = 0
			}

			formattedPath := fileColor.Format(path)
			if data.Bool(fileOnlyFlag.Name()) {
				output.Stdout(formattedPath)
				break
			}

			if data.Bool(hideFileFlag.Name()) {
				for list.length > 0 {
					output.Stdout(list.pop())
				}
				output.Stdout(formattedString)
			} else {
				for list.length > 0 {
					output.Stdout("%s:%s", formattedPath, list.pop())
				}
				output.Stdout("%s:%s", formattedPath, formattedString)
			}
		}

		return nil
	})
}

type element struct {
	value string
	next  *element
}

type linkedList struct {
	front  *element
	back   *element
	length int
}

func (ll *linkedList) pushBack(s string) {
	newEl := &element{
		value: s,
	}
	if ll.length == 0 {
		ll.front = newEl
		ll.back = newEl
	} else {
		ll.back.next = newEl
		ll.back = newEl
	}
	ll.length++
}

func (ll *linkedList) pop() string {
	r := ll.front.value
	if ll.length == 1 {
		ll.front = nil
		ll.back = nil
	} else {
		ll.front = ll.front.next
	}
	ll.length--
	return r
}
