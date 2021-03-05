package grep

import (
	"bufio"
	"encoding/json"
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

	fileArg = commands.StringFlag("file", 'f', nil)
	// Don't show file names
	hideFileFlag = commands.BoolFlag("hideFile", 'h')
	// Only show file names (hide lines).
	fileOnlyFlag = commands.BoolFlag("fileOnly", 'l')
	// Show the matched line and the `n` lines before it.
	beforeFlag = commands.IntFlag("before", 'b', nil)
	// Show the matched line and the `n` lines after it.
	afterFlag = commands.IntFlag("after", 'a', nil)
	// Directory flag to search through an aliased directory instead of pwd.
	dirFlag = commands.StringFlag("directory", 'd', nil /*todo completor*/)
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

type recursive struct {
	DirectoryAliases map[string]string
	changed          bool
}

func (*recursive) Name() string             { return "recursive-grep" }
func (*recursive) Alias() string            { return "rp" }
func (*recursive) Option() *commands.Option { return nil }
func (*recursive) Flags() []commands.Flag {
	return []commands.Flag{
		fileArg,
		hideFileFlag,
		fileOnlyFlag,
		beforeFlag,
		afterFlag,
		dirFlag,
	}
}

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

func (*recursive) Subcommands() map[string]commands.Command {
	return nil
}

func (r *recursive) Process(cos commands.CommandOS, args, flags map[string]*commands.Value, _ *commands.OptionInfo, ffs filterFuncs) (*commands.ExecutorResponse, bool) {
	hideFile := flags[hideFileFlag.Name()].Bool() != nil && *flags[hideFileFlag.Name()].Bool()
	fileOnly := flags[fileOnlyFlag.Name()].Bool() != nil && *flags[fileOnlyFlag.Name()].Bool()
	var linesBefore, linesAfter int
	if intPtr := flags[beforeFlag.Name()].Int(); intPtr != nil {
		linesBefore = *intPtr
	}
	if intPtr := flags[afterFlag.Name()].Int(); intPtr != nil {
		linesAfter = *intPtr
	}

	var fr *regexp.Regexp
	if fileRegex := flags["file"].String(); fileRegex != nil {
		var err error
		if fr, err = regexp.Compile(*fileRegex); err != nil {
			cos.Stderr("invalid filename regex: %v", err)
			return nil, false
		}
	}

	dir := startDir
	if strPtr := flags[dirFlag.Name()].String(); strPtr != nil {
		var ok bool
		dir, ok = r.DirectoryAliases[*strPtr]
		if !ok {
			cos.Stderr("unknown alias: %q", *strPtr)
			return nil, false
		}
	}

	err := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
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

		list := &linkedList{}
		linesSinceMatch := linesAfter
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			formattedString, ok := ffs.Apply(scanner.Text())
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
			if fileOnly {
				cos.Stdout(formattedPath)
				break
			}

			if hideFile {
				for list.length > 0 {
					cos.Stdout(list.pop())
				}
				cos.Stdout(formattedString)
			} else {
				for list.length > 0 {
					cos.Stdout("%s:%s", formattedPath, list.pop())
				}
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
