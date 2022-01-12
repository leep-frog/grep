package grep

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/color"
)

var (
	startDir                                   = "."
	osOpen   func(s string) (io.Reader, error) = func(s string) (io.Reader, error) { return os.Open(s) }

	ignoreFilePattern = command.StringListNode("IGNORE_PATTERN", "Files that match these will be ignored", 1, command.UnboundedList, command.ListIsRegex())

	fileArg       = command.StringFlag("file", 'f', "Only select files that match this pattern")
	invertFileArg = command.StringFlag("invert-file", 'F', "Only select files that don't match this pattern")
	hideFileFlag  = command.BoolFlag("hide-file", 'h', "Don't show file names")
	fileOnlyFlag  = command.BoolFlag("file-only", 'l', "Only show file names")
	beforeFlag    = command.IntFlag("before", 'b', "Show the matched line and the n lines before it")
	afterFlag     = command.IntFlag("after", 'a', "Show the matched line and the n lines after it")
	dirFlag       = command.StringFlag("directory", 'd', "Search through the provided directory instead of pwd", &command.Completor{
		SuggestionFetcher: &command.FileFetcher{
			IgnoreFiles: true,
		},
	})
	hideLineFlag = command.BoolFlag("hide-lines", 'n', "Don't include the line number in the output")

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
		InputSource: &recursive{},
	}
}

type recursive struct {
	// TODO: add way to get this from other command
	DirectoryAliases   map[string]string
	IgnoreFilePatterns map[string]bool
	changed            bool
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

// TODO: Use store CLI for this
func (r *recursive) addIgnorePattern(output command.Output, data *command.Data) {
	if r.IgnoreFilePatterns == nil {
		r.IgnoreFilePatterns = map[string]bool{}
	}
	for _, pattern := range data.StringList(ignoreFilePattern.Name()) {
		r.IgnoreFilePatterns[pattern] = true
	}
	r.changed = true
}

func (r *recursive) deleteIgnorePattern(output command.Output, data *command.Data) {
	if r.IgnoreFilePatterns == nil {
		return
	}
	for _, pattern := range data.StringList(ignoreFilePattern.Name()) {
		delete(r.IgnoreFilePatterns, pattern)
	}
	r.changed = true
}

func (r *recursive) listIgnorePattern(output command.Output, data *command.Data) {
	var patterns []string
	for p := range r.IgnoreFilePatterns {
		patterns = append(patterns, p)
	}
	sort.Strings(patterns)
	for _, p := range patterns {
		output.Stdout(p)
	}
}

func (r *recursive) MakeNode(n *command.Node) *command.Node {
	f := &command.Completor{
		SuggestionFetcher: command.SimpleFetcher(func(v *command.Value, d *command.Data) (*command.Completion, error) {
			var s []string
			for p := range r.IgnoreFilePatterns {
				s = append(s, p)
			}
			return &command.Completion{
				Suggestions: s,
			}, nil
		}),
	}
	return command.BranchNode(map[string]*command.Node{
		"if": command.SerialNodesTo(command.BranchNode(map[string]*command.Node{
			"a": command.SerialNodes(
				command.Description("Add a global file ignore pattern"),
				ignoreFilePattern,
				command.ExecutorNode(r.addIgnorePattern),
			),
			"d": command.SerialNodes(
				command.Description("Deletes a global file ignore pattern"),
				ignoreFilePattern.AddOptions(f),
				command.ExecutorNode(r.deleteIgnorePattern),
			),
			"l": command.SerialNodes(
				command.Description("List global file ignore patterns"),
				command.ExecutorNode(r.listIgnorePattern),
			),
		}, nil), command.Description("Commands around global ignore file patterns")),
	}, n)
}

func (r *recursive) Changed() bool {
	return r.changed
}

func (r *recursive) Process(output command.Output, data *command.Data, fltr filter) error {
	var nameRegexes []*regexp.Regexp
	for ifp := range r.IgnoreFilePatterns {
		// ListIsRegex ArgOption ensures that these regexes are valid, so it's okay to use MustCompile here.
		nameRegexes = append(nameRegexes, regexp.MustCompile(ifp))
	}
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

		for _, r := range nameRegexes {
			if r.MatchString(fi.Name()) {
				return nil
			}
		}

		f, err := osOpen(path)
		if err != nil {
			return output.Stderrf("failed to open file %q: %v", path, err)
		}

		scanner := bufio.NewScanner(f)
		list := newLinkedList(fltr, data, scanner)
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
	filter filter

	data *command.Data

	// lastMatch contains how many lines ago a match was found.
	lastMatch int
	scanner   *bufio.Scanner

	clearBefores bool
}

func newLinkedList(fltr filter, data *command.Data, scanner *bufio.Scanner) *linkedList {
	return &linkedList{
		before:  data.Int(beforeFlag.Name()),
		after:   data.Int(afterFlag.Name()),
		filter:  fltr,
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
		if formattedString, ok := apply(ll.filter, s, ll.data); ok {
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
