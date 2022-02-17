package grep

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/color"
)

var (
	patternArgName = "PATTERN"
	patternArg     = command.StringListListNode(patternArgName, "Pattern(s) required to be present in each line. The list breaker acts as an OR operator for groups of regexes", "|", 0, command.UnboundedList, command.ValidatorList(command.IsRegex()))
	caseFlag       = command.BoolFlag("case", 'i', "Don't ignore character casing")
	invertFlag     = command.NewListFlag[string]("invert", 'v', "Pattern(s) required to be absent in each line", 0, command.UnboundedList, command.ValidatorList(command.IsRegex()))
	matchOnlyFlag  = command.BoolFlag("match-only", 'o', "Only show the matching segment")

	matchColor = &color.Format{
		Color:     color.Green,
		Thickness: color.Bold,
	}
)

type filter interface {
	filter(string) ([]*match, bool)
}

type andFilter struct {
	filters []filter
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

func apply(f filter, s string, data *command.Data) (string, bool) {
	matchOnly := data.Bool(matchOnlyFlag.Name())
	otherString := s

	matches, ok := f.filter(s)
	if !ok {
		return "", false
	}
	matches = disjointMatches(matches)

	var offset int
	var mo []string
	for _, m := range matches {
		if matchOnly {
			mo = append(mo, otherString[m.start:m.end])
		} else {
			origLen := len(otherString)
			otherString = fmt.Sprintf(
				"%s%s%s",
				otherString[:(offset+m.start)],
				matchColor.Format(otherString[(offset+m.start):(offset+m.end)]),
				otherString[(offset+m.end):],
			)
			offset += len(otherString) - origLen
		}
	}

	if matchOnly {
		return strings.Join(mo, "..."), true
	}
	return otherString, true
}

type inputSource interface {
	Name() string
	Process(command.Output, *command.Data, filter) error
	Flags() []command.Flag
	MakeNode(*command.Node) *command.Node
	Setup() []string
	Changed() bool
}

type Grep struct {
	InputSource inputSource
}

func (g *Grep) Load(jsn string) error {
	if jsn == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(jsn), g); err != nil {
		return fmt.Errorf("failed to unmarshal json for grep object: %v", err)
	}
	return nil
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

func (cm *colorMatcher) filter(s string) ([]*match, bool) {
	indices := cm.r.FindStringIndex(s)
	if indices == nil {
		return nil, false
	}
	return []*match{&match{
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

/*func colorMatch(r *regexp.Regexp) func(string) (*match, bool) {
	return func(s string) (*match, bool) {
		indices := r.FindStringIndex(s)
		if indices == nil {
			return nil, false
		}
		return &match{
			start: indices[0],
			end:   indices[1],
		}, true
	}
}*/
func colorMatch(r *regexp.Regexp) filter {
	return &colorMatcher{r}
}

func (g *Grep) Complete(*command.Input, *command.Data) (*command.Completion, error) {
	return nil, nil
}

func (g *Grep) Execute(output command.Output, data *command.Data) error {
	ignoreCase := !data.Bool(caseFlag.Name())

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

func (g *Grep) Node() *command.Node {
	flags := append(g.InputSource.Flags(), caseFlag, invertFlag, matchOnlyFlag)
	flagNode := command.NewFlagNode(flags...)

	return g.InputSource.MakeNode(command.SerialNodes(flagNode, patternArg, command.ExecuteErrNode(g.Execute)))
}
