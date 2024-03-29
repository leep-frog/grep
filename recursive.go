package grep

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/leep-frog/command/color"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
)

var (
	startDir = "."
	osOpen   = func(s string) (io.Reader, error) { return os.Open(s) }

	ignoreFilePattern = commander.ListArg[string]("IGNORE_PATTERN", "Files that match these will be ignored", 1, command.UnboundedList, commander.ListifyValidatorOption(commander.IsRegex()))

	ignoreIgnoreFiles = commander.BoolFlag("ignore-ignore-files", 'x', "Ignore the provided IGNORE_PATTERNS")
	fileArg           = commander.Flag[string]("file", 'f', "Only select files that match this pattern")
	invertFileArg     = commander.Flag[string]("invert-file", 'F', "Only select files that don't match this pattern")
	hideFileFlag      = commander.BoolFlag("hide-file", 'h', "Don't show file names")
	fileOnlyFlag      = commander.BoolFlag("file-only", 'l', "Only show file names")
	beforeFlag        = commander.Flag[int]("before", 'b', "Show the matched line and the n lines before it")
	afterFlag         = commander.Flag[int]("after", 'a', "Show the matched line and the n lines after it")
	depthFlag         = commander.Flag[int]("depth", 'd', "The depth of files to search", commander.NonNegative[int]())
	dirFlag           = commander.Flag[string]("directory", 'D', "Search through the provided directory instead of pwd", &commander.FileCompleter[string]{IgnoreFiles: true})
	hideLineFlag      = commander.BoolFlag("hide-lines", 'n', "Don't include the line number in the output")
	wholeFile         = commander.BoolFlag("whole-file", 'w', "Whether or not to search the whole file (i.e. multi-wrap searching) in one regex")

	fileColor = color.Yellow

	lineColor = color.Cyan
)

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

func (*recursive) Name() string {
	return "rp"
}

func (*recursive) Setup() []string { return nil }
func (*recursive) Flags() []commander.FlagInterface {
	return []commander.FlagInterface{
		fileArg,
		invertFileArg,
		hideFileFlag,
		fileOnlyFlag,
		beforeFlag,
		afterFlag,
		depthFlag,
		dirFlag,
		hideLineFlag,
		ignoreIgnoreFiles,
	}
}

func (r *recursive) addIgnorePattern(output command.Output, data *command.Data) error {
	if r.IgnoreFilePatterns == nil {
		r.IgnoreFilePatterns = map[string]bool{}
	}
	for _, pattern := range data.StringList(ignoreFilePattern.Name()) {
		r.IgnoreFilePatterns[pattern] = true
	}
	r.changed = true
	return nil
}

func (r *recursive) deleteIgnorePattern(output command.Output, data *command.Data) error {
	if r.IgnoreFilePatterns == nil {
		return nil
	}
	for _, pattern := range data.StringList(ignoreFilePattern.Name()) {
		delete(r.IgnoreFilePatterns, pattern)
	}
	r.changed = true
	return nil
}

func (r *recursive) listIgnorePattern(output command.Output, data *command.Data) error {
	var patterns []string
	for p := range r.IgnoreFilePatterns {
		patterns = append(patterns, p)
	}
	sort.Strings(patterns)
	for _, p := range patterns {
		output.Stdoutln(p)
	}
	return nil
}

func (r *recursive) MakeNode(n command.Node) command.Node {
	f := commander.CompleterFromFunc(func(v []string, d *command.Data) (*command.Completion, error) {
		var s []string
		for p := range r.IgnoreFilePatterns {
			s = append(s, p)
		}
		return &command.Completion{
			Suggestions: s,
		}, nil
	})
	return &commander.BranchNode{
		Branches: map[string]command.Node{
			"if": commander.SerialNodes(
				commander.Description("Commands around global ignore file patterns"),
				&commander.BranchNode{
					Branches: map[string]command.Node{
						"a": commander.SerialNodes(
							commander.Description("Add a global file ignore pattern"),
							ignoreFilePattern,
							&commander.ExecutorProcessor{F: r.addIgnorePattern},
						),
						"d": commander.SerialNodes(
							commander.Description("Deletes a global file ignore pattern"),
							ignoreFilePattern.AddOptions(f),
							&commander.ExecutorProcessor{F: r.deleteIgnorePattern},
						),
						"l": commander.SerialNodes(
							commander.Description("List global file ignore patterns"),
							&commander.ExecutorProcessor{F: r.listIgnorePattern},
						),
					},
				},
			),
		},
		Default: n,
	}
}

func (r *recursive) Changed() bool {
	return r.changed
}

func (r *recursive) Process(output command.Output, data *command.Data, fltr filter, ss *sliceSet) error {
	var nameRegexes []*regexp.Regexp

	if !ignoreIgnoreFiles.Get(data) {
		for ifp := range r.IgnoreFilePatterns {
			// ListIsRegex ArgumentOption ensures that these regexes are valid, so it's okay to use MustCompile here.
			nameRegexes = append(nameRegexes, regexp.MustCompile(ifp))
		}
	}
	var fr *regexp.Regexp

	if data.Has(fileArg.Name()) {
		f := data.String(fileArg.Name())
		var err error
		if fr, err = regexp.Compile(f); err != nil {
			return output.Stderrf("invalid filename regex: %v\n", err)
		}
	}

	var ifr *regexp.Regexp
	if data.Has(invertFileArg.Name()) {
		f := data.String(invertFileArg.Name())
		var err error
		if ifr, err = regexp.Compile(f); err != nil {
			return output.Stderrf("invalid invert filename regex: %v\n", err)
		}
	}

	dir := startDir
	if data.Has(dirFlag.Name()) {
		da := data.String(dirFlag.Name())
		var ok bool
		dir, ok = r.DirectoryAliases[da]
		if !ok {
			return output.Stderrf("unknown alias: %q\n", da)
		}
	}

	maxDepth := depthFlag.GetOrDefault(data, 0)

	return filepath.WalkDir(dir, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return output.Stderrf("file not found: %s\n", path)
			}
			return output.Stderrf("failed to access path %q: %v\n", path, err)
		}

		if de.IsDir() {
			if maxDepth > 0 && strings.Count(path, string(os.PathSeparator)) >= maxDepth {
				return fs.SkipDir
			}
			return nil
		}

		if fr != nil && !fr.MatchString(de.Name()) {
			return nil
		}

		if ifr != nil && ifr.MatchString(de.Name()) {
			return nil
		}

		for _, r := range nameRegexes {
			if r.MatchString(de.Name()) {
				return nil
			}
		}

		f, err := osOpen(path)
		if err != nil {
			return output.Stderrf("failed to open file %q: %v\n", path, err)
		}

		scanner := bufio.NewScanner(f)
		list := newLinkedList(fltr, data, scanner)
		for formattedString, line, ok := list.getNext(ss); ok; formattedString, line, ok = list.getNext(ss) {
			if data.Bool(fileOnlyFlag.Name()) {
				applyFormatWithColor(output, data, fileColor, []string{"", path})
				output.Stdoutln()
				break
			}

			var needColon bool
			if !data.Bool(hideFileFlag.Name()) {
				applyFormatWithColor(output, data, fileColor, []string{"", path})
				needColon = true
			}
			if !data.Bool(hideLineFlag.Name()) {
				if needColon {
					output.Stdout(":")
				} else {
					needColon = true
				}
				applyFormatWithColor(output, data, lineColor, []string{"", fmt.Sprintf("%d", line)})
			}
			if needColon {
				output.Stdout(":")
			}
			applyFormat(output, data, formattedString)
			output.Stdoutln()
		}

		return nil
	})
}

type element struct {
	value []string
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

func (ll *linkedList) getNext(ss *sliceSet) ([]string, int, bool) {
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
			return nil, 0, false
		}
		s := ll.scanner.Text()

		// If we got a match, then update lastMatch and print this line and any previous ones.
		if formattedString, ok := apply(ll.filter, s, ll.data, ss); ok {
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
			return []string{s}, ll.lineCount, true
		}

		// Otherwise, we store the string in our behind list incase
		// we get a match later.
		ll.pushBack([]string{s}, ll.lineCount)
		if ll.length > ll.before {
			ll.pop()
		}
	}
}

func (ll *linkedList) pushBack(ss []string, i int) {
	newEl := &element{
		value: ss,
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

func (ll *linkedList) pop() ([]string, int) {
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
