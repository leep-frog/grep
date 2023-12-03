package grep

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/leep-frog/command/color"
	"github.com/leep-frog/command/command"
	"github.com/leep-frog/command/commander"
)

var (
	defaultColorValue = len(os.Getenv("LEEP_FROG_RP_NO_COLOR")) == 0
	patternArgName    = "PATTERN"
	patternArg        = commander.StringListListProcessor(patternArgName, "Pattern(s) required to be present in each line. The list breaker acts as an OR operator for groups of regexes", "|", 0, command.UnboundedList, commander.ListifyValidatorOption(commander.IsRegex()))
	caseFlag          = commander.BoolFlag("case", 'i', "Don't ignore character casing")
	wholeWordFlag     = commander.BoolFlag("whole-word", 'w', "Whether or not to search for exact match")
	invertFlag        = commander.ListFlag[string]("invert", 'v', "Pattern(s) required to be absent in each line", 0, command.UnboundedList, commander.ListifyValidatorOption(commander.IsRegex()))
	matchOnlyFlag     = commander.BoolFlag("match-only", 'o', "Only show the matching segment")
	colorFlag         = commander.BoolFlag("color", 'C', "Force (or unforce) the grep output to include color")

	matchColor = color.MultiFormat(color.Text(color.Green), color.Bold())
)

func shouldColor(data *command.Data) bool {
	if data.Has(colorFlag.Name()) && colorFlag.Get(data) {
		return !defaultColorValue
	}
	return defaultColorValue
}

type filter interface {
	filter(string) ([]*match, bool)
	fmt.Stringer
}

type andFilter struct {
	filters []filter
}

func (af *andFilter) String() string {
	var r []string
	for _, f := range af.filters {
		r = append(r, fmt.Sprintf("(%s)", f.String()))
	}
	return strings.Join(r, " && ")
}

func (af *andFilter) filter(s string) ([]*match, bool) {
	var ms []*match
	for _, f := range af.filters {
		if m, ok := f.filter(s); ok {
			ms = append(ms, m...)
		} else {
			return nil, false
		}
	}
	return ms, true
}

type orFilter struct {
	filters []filter
}

func (of *orFilter) String() string {
	var r []string
	for _, f := range of.filters {
		r = append(r, fmt.Sprintf("(%s)", f.String()))
	}
	return strings.Join(r, " || ")
}

func (of *orFilter) filter(s string) ([]*match, bool) {
	for _, f := range of.filters {
		if m, ok := f.filter(s); ok {
			return m, true
		}
	}
	return nil, false
}

// Used to determine how to color overlapping matches
type event struct {
	start bool
	idx   int
}

func disjointMatches(ms []*match) []*match {
	events := make([]*event, 0, 2*len(ms))
	for _, m := range ms {
		events = append(events, &event{start: true, idx: m.start}, &event{idx: m.end})
	}
	sort.Slice(events, func(i, j int) bool {
		ie := events[i]
		je := events[j]
		return ie.idx < je.idx || (ie.idx == je.idx && ie.start)
	})

	var ums []*match
	var inMatchCount int
	var newStart int
	for _, e := range events {
		if e.start {
			inMatchCount++
			if inMatchCount == 1 {
				newStart = e.idx
			}
		} else {
			inMatchCount--
			if inMatchCount == 0 {
				ums = append(ums, &match{start: newStart, end: e.idx})
			}
		}
	}
	return ums
}

// applyFormat should be called on the response returned from the `apply` method
func applyFormat(o command.Output, d *command.Data, ss []string) {
	applyFormatWithColor(o, d, matchColor, ss)
}

func applyFormatWithColor(o command.Output, d *command.Data, f *color.Format, ss []string) {
	if !shouldColor(d) {
		o.Stdout(strings.Join(ss, ""))
		return
	}

	for i, s := range ss {
		if i%2 == 1 {
			f.Apply(o)
		}
		o.Stdout(s)
		if i%2 == 1 {
			color.Init().Apply(o)
		}
	}
}

// Returns a string slice of [Not-match, match, not-match, match, ..., not-match]
func apply(f filter, s string, data *command.Data) ([]string, bool) {
	matchOnly := data.Bool(matchOnlyFlag.Name())

	matches, ok := f.filter(s)
	if !ok {
		return nil, false
	}

	// Check if it's a full line match.
	if len(matches) == 0 {
		// If it's a full line match, then no need to provide formatting.
		return []string{s}, true
	}
	matches = disjointMatches(matches)

	var mo []string

	if matchOnly {
		mo = append(mo, "")
	} else {
		mo = append(mo, s[:matches[0].start])
	}

	for idx, m := range matches {
		if matchOnly {
			mo = append(mo, s[m.start:m.end])
			if idx != len(matches)-1 {
				mo = append(mo, "...")
			}
		} else {
			mo = append(mo, s[m.start:m.end])
			if idx == len(matches)-1 {
				mo = append(mo, s[m.end:])
			} else {
				mo = append(mo, s[m.end:matches[idx+1].start])
			}
		}
	}

	if matchOnly {
		return []string{strings.Join(mo, "")}, true
	}

	return mo, true
}

type inputSource interface {
	Name() string
	Process(command.Output, *command.Data, filter) error
	Flags() []commander.FlagInterface
	MakeNode(command.Node) command.Node
	Setup() []string
	Changed() bool
}

type Grep struct {
	InputSource inputSource
}

func (g *Grep) Changed() bool {
	return g.InputSource.Changed()
}

func (g *Grep) Name() string {
	return g.InputSource.Name()
}

func (g *Grep) Setup() []string {
	return g.InputSource.Setup()
}

type match struct {
	start int
	end   int
}

type colorMatcher struct {
	r *regexp.Regexp
}

func (cm *colorMatcher) String() string {
	return fmt.Sprintf("COLOR{%s}", cm.r.String())
}

func (cm *colorMatcher) filter(s string) ([]*match, bool) {
	indices := cm.r.FindStringIndex(s)
	if indices == nil {
		return nil, false
	}
	return []*match{{
		start: indices[0],
		end:   indices[1],
	}}, true
}

type invertMatcher struct {
	r *regexp.Regexp
}

func (im *invertMatcher) filter(s string) ([]*match, bool) {
	return nil, !im.r.MatchString(s)
}

func (im *invertMatcher) String() string {
	return fmt.Sprintf("![%s]", im.r.String())
}

func colorMatch(r *regexp.Regexp) filter {
	return &colorMatcher{r}
}

func (g *Grep) Complete(*command.Input, *command.Data) (*command.Completion, error) {
	return nil, nil
}

func (g *Grep) Execute(output command.Output, data *command.Data) error {
	ignoreCase := !caseFlag.Get(data)
	wholeWord := wholeWordFlag.Get(data)

	var filters []filter
	ps := data.Values[patternArgName]
	if ps != nil {
		of := &orFilter{}
		for _, patternGroup := range command.GetData[[][]string](data, patternArgName) {
			af := &andFilter{}
			for _, pattern := range patternGroup {
				if ignoreCase {
					pattern = fmt.Sprintf("(?i)%s", pattern)
				}
				if wholeWord {
					pattern = fmt.Sprintf("\\b%s\\b", pattern)
				}
				// ListIsRegex ensures that only valid regexes reach this point.
				af.filters = append(af.filters, colorMatch(regexp.MustCompile(pattern)))
			}
			of.filters = append(of.filters, af)
		}
		if len(of.filters) > 0 {
			filters = append(filters, of)
		}
	}

	for _, pattern := range data.StringList(invertFlag.Name()) {
		// ListIsRegex ensures that only valid regexes reach this point.
		r := regexp.MustCompile(pattern)
		filters = append(filters, &invertMatcher{r})
	}

	return g.InputSource.Process(output, data, &andFilter{filters})
}

func (g *Grep) Node() command.Node {
	flags := append(g.InputSource.Flags(),
		caseFlag,
		colorFlag,
		invertFlag,
		matchOnlyFlag,
		wholeWordFlag,
	)
	flagProcessor := commander.FlagProcessor(flags...)

	return g.InputSource.MakeNode(commander.SerialNodes(flagProcessor, patternArg, &commander.ExecutorProcessor{F: g.Execute}))
}
