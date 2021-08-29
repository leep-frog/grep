package grep

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

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
	// Don't include the line number in the output.
	hideLineFlag = command.BoolFlag("hideLines", 'n')

	fileColor = &color.Format{
		Color: color.Yellow,
	}

	lineColor = &color.Format{
		Color: color.Cyan,
	}
)

func colorLine(n int) string {
	return lineColor.Format(strconv.Itoa(n))
}

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
		hideLineFlag,
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
			return output.Stderrf("invalid filename regex: %v", err)
		}
	}

	var ifr *regexp.Regexp
	if data.HasArg(invertFileArg.Name()) {
		f := data.String(invertFileArg.Name())
		var err error
		if ifr, err = regexp.Compile(f); err != nil {
			return output.Stderrf("invalid invert filename regex: %v", err)
		}
	}

	dir := startDir
	if data.HasArg(dirFlag.Name()) {
		da := data.String(dirFlag.Name())
		var ok bool
		dir, ok = r.DirectoryAliases[da]
		if !ok {
			return output.Stderrf("unknown alias: %q", da)
		}
	}

	return filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return output.Stderrf("file not found: %s", path)
			}
			return output.Stderrf("failed to access path %q: %v", path, err)
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
			return output.Stderrf("failed to open file %q: %v", path, err)
		}

		scanner := bufio.NewScanner(f)
		list := newLinkedList(ffs, data, scanner)
		for formattedString, line, ok := list.getNext(); ok; formattedString, line, ok = list.getNext() {
			formattedPath := fileColor.Format(path)
			if data.Bool(fileOnlyFlag.Name()) {
				output.Stdout(formattedPath)
				break
			}

			var parts []string
			if !data.Bool(hideFileFlag.Name()) {
				parts = append(parts, formattedPath)
			}
			if !data.Bool(hideLineFlag.Name()) {
				parts = append(parts, colorLine(line))
			}
			parts = append(parts, formattedString)
			output.Stdout(strings.Join(parts, ":"))
		}

		return nil
	})
}

type element struct {
	value string
	n     int
	next  *element
}

type linkedList struct {
	front  *element
	back   *element
	length int

	lineCount int

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

func (ll *linkedList) getNext() (string, int, bool) {
	for {
		// If we have lines to print, then just return the lines.
		if ll.clearBefores {
			if ll.length > 0 {
				s, i := ll.pop()
				return s, i, true
			}
			ll.clearBefores = false
		}

		// Otherwise, look for lines to return.
		ll.lineCount++
		if !ll.scanner.Scan() {
			return "", 0, false
		}
		s := ll.scanner.Text()

		// If we got a match, then update lastMatch and print this line and any previous ones.
		if formattedString, ok := ll.ffs.Apply(s, ll.data); ok {
			ll.lastMatch = 0
			ll.pushBack(formattedString, ll.lineCount)
			ll.clearBefores = true
			continue
		}

		// Otherwise, increment lastMatch.
		ll.lastMatch++

		// If we are still in the "after" window from our last match,
		// then we want to print out this line.
		if ll.lastMatch <= ll.after {
			return s, ll.lineCount, true
		}

		// Otherwise, we store the string in our behind list incase
		// we get a match later.
		ll.pushBack(s, ll.lineCount)
		if ll.length > ll.before {
			ll.pop()
		}
	}
}

func (ll *linkedList) pushBack(s string, i int) {
	newEl := &element{
		value: s,
		n:     i,
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

func (ll *linkedList) pop() (string, int) {
	r := ll.front.value
	i := ll.front.n
	if ll.length == 1 {
		ll.front = nil
		ll.back = nil
	} else {
		ll.front = ll.front.next
	}
	ll.length--
	return r, i
}
