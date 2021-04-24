package grep

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/color"
)

var (
	patternArgName = "pattern"
	patternArg     = command.StringListNode(patternArgName, 0, -1, nil)
	caseFlagName   = "ignoreCase"
	caseFlag       = command.BoolFlag(caseFlagName, 'i')
	invertFlagName = "invert"
	invertFlag     = command.StringListFlag(invertFlagName, 'v', 0, command.UnboundedList, nil)
	// TODO: or pattern

	matchColor = &color.Format{
		Color:     color.Green,
		Thickness: color.Bold,
	}
)

type filterFunc func(string) (*match, bool)

type filterFuncs []filterFunc

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

func (ffs filterFuncs) Apply(s string) (string, bool) {
	otherString := s

	var matches []*match
	for _, ff := range ffs {
		m, ok := ff(s)
		if !ok {
			return "", false
		}
		if m != nil {
			matches = append(matches, m)
		}
	}
	matches = disjointMatches(matches)

	var offset int
	for _, m := range matches {
		origLen := len(otherString)
		otherString = fmt.Sprintf(
			"%s%s%s",
			otherString[:(offset+m.start)],
			matchColor.Format(otherString[(offset+m.start):(offset+m.end)]),
			otherString[(offset+m.end):],
		)
		offset += len(otherString) - origLen
	}

	return otherString, true
}

type inputSource interface {
	Name() string
	Process(command.Output, *command.Data, filterFuncs) error
	Flags() []command.Flag
	Setup() []string
	Changed() bool
	Load(string) error
}

type Grep struct {
	caseSensitive bool
	inputSource   inputSource
}

func (g *Grep) Load(jsn string) error {
	return g.inputSource.Load(jsn)
}

func (g *Grep) Changed() bool {
	return g.inputSource.Changed()
}

func (g *Grep) Name() string {
	return g.inputSource.Name()
}

func (g *Grep) Setup() []string {
	return g.inputSource.Setup()
}

type match struct {
	start int
	end   int
}

func colorMatch(r *regexp.Regexp) func(string) (*match, bool) {
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
}

func (g *Grep) Complete(*command.Input, *command.Data) *command.CompleteData {
	// Currently no way to autocomplete regular expressions.
	return nil
}

func (g *Grep) Execute(output command.Output, data *command.Data) error {
	ignoreCase := data.Values[caseFlagName].Bool()

	var ffs filterFuncs //[]func(string) (*formatter, bool)
	for _, pattern := range data.Values[patternArgName].StringList() {
		if ignoreCase {
			pattern = fmt.Sprintf("(?i)%s", pattern)
		}
		r, err := regexp.Compile(pattern)
		if err != nil {
			return output.Stderr("invalid regex: %v", err)
		}
		ffs = append(ffs, colorMatch(r))
	}

	for _, pattern := range data.Values[invertFlagName].StringList() {
		r, err := regexp.Compile(pattern)
		if err != nil {
			return output.Stderr("invalid invert regex: %v", err)
		}
		ffs = append(ffs, func(s string) (*match, bool) { return nil, !r.MatchString(s) })
	}

	return g.inputSource.Process(output, data, ffs)
}

func (g *Grep) Node() *command.Node {
	flags := append(g.inputSource.Flags(), caseFlag, invertFlag)
	flagNode := command.NewFlagNode(flags...)
	return command.SerialNodes(
		flagNode,
		patternArg,
		command.ExecutorNode(g.Execute),
		//command.ExecutorNode(g.P,
	)
}

/*func (g *Grep) Node() command.NodeProcessor {
	flags := append(g.inputSource.Flags(), caseFlag, invertFlag)
	return &command.Node{
		Processor: command.ExecutorNode(g.),
	}
}*/

/*func (g *Grep) Command() command.Command {
	flags := []command.Flag{
		caseFlag,
		invertFlag,
	}
	return &command.CommandBranch{
		Subcommands: g.inputSource.Subcommands(),
		TerminusCommand: &command.TerminusCommand{
			Executor: g.execute,
			Args: []command.Arg{
				patternArg,
			},
			Flags: append(flags, g.inputSource.Flags()...),
		},
	}
}*/
