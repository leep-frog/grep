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

		scanner := bufio.NewScanner(f)
		list := newLinkedList(ffs, data, scanner)
		for formattedString, ok := list.getNext(); ok; formattedString, ok = list.getNext() {
			formattedPath := fileColor.Format(path)
			if data.Bool(fileOnlyFlag.Name()) {
				output.Stdout(formattedPath)
				break
			}

			if data.Bool(hideFileFlag.Name()) {
				output.Stdout(formattedString)
			} else {
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

	before int
	after  int
	ffs    filterFuncs

	data *command.Data

	// lastMatch contains how many lines ago a match was found.
	lastMatch int
	scanner   *bufio.Scanner

	clearBefores bool
}

func newLinkedList(ffs filterFuncs, data *command.Data, scanner *bufio.Scanner) *linkedList {
	return &linkedList{
		before:  data.Int(beforeFlag.Name()),
		after:   data.Int(afterFlag.Name()),
		ffs:     ffs,
		scanner: scanner,

		data: data,

		lastMatch: data.Int(afterFlag.Name()),
	}
}

func (ll *linkedList) getNext() (string, bool) {
	for {
		// If we have lines to print, then just return the lines.
		if ll.clearBefores {
			if ll.length > 0 {
				return ll.pop(), true
			}
			ll.clearBefores = false
		}

		// Otherwise, look for lines to return.
		if !ll.scanner.Scan() {
			return "", false
		}
		s := ll.scanner.Text()

		// If we got a match, then update lastMatch and print this line and any previous ones.
		if formattedString, ok := ll.ffs.Apply(s, ll.data); ok {
			ll.lastMatch = 0
			ll.pushBack(formattedString)
			ll.clearBefores = true
			continue
		}

		// Otherwise, increment lastMatch.
		ll.lastMatch++

		// If we are still in the "after" window from our last match,
		// then we want to print out this line.
		if ll.lastMatch <= ll.after {
			return s, true
		}

		// Otherwise, we store the string in our behind list incase
		// we get a match later.
		ll.pushBack(s)
		if ll.length > ll.before {
			ll.pop()
		}
	}
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
