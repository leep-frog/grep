package grep

import (
	"fmt"
	"regexp"

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

type filterFunc func(string) (*formatter, bool)

type filterFuncs []filterFunc

func (ffs filterFuncs) Apply(s string) (string, bool) {
	otherString := s
	for _, ff := range ffs {
		ft, ok := ff(s)
		if !ok {
			return "", false
		}

		if ft != nil {
			idx, jdx := ft.indices[0], ft.indices[1]
			otherString = fmt.Sprintf("%s%s%s", otherString[:idx], ft.format.Format(otherString[idx:jdx]), otherString[jdx:])
		}
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

type formatter struct {
	indices []int
	format  *color.Format
}

func colorMatch(r *regexp.Regexp) func(string) (*formatter, bool) {
	return func(s string) (*formatter, bool) {
		indices := r.FindStringIndex(s)
		if indices == nil {
			return nil, false
		}
		return &formatter{
			indices: indices,
			format:  matchColor,
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
		ffs = append(ffs, func(s string) (*formatter, bool) { return nil, !r.MatchString(s) })
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
