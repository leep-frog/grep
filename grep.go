package grep

import (
	"fmt"
	"regexp"

	"github.com/leep-frog/commands/color"
	"github.com/leep-frog/commands/commands"
)

var (
	patternArgName = "pattern"
	patternArg     = commands.StringListNode(patternArgName, 0, -1, nil)
	caseFlagName   = "ignoreCase"
	caseFlag       = commands.BoolFlag(caseFlagName, 'i')
	invertFlagName = "invert"
	invertFlag     = commands.StringListFlag(invertFlagName, 'v', 0, commands.UnboundedList, nil)
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
	Alias() string
	Process(*commands.WorldState, filterFuncs) bool
	Flags() []commands.Flag
	Option() *commands.Option
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

func (g *Grep) Alias() string {
	return g.inputSource.Alias()
}

func (g *Grep) Option() *commands.Option {
	return g.inputSource.Option()
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

func (g *Grep) Complete(ws *commands.WorldState) bool {
	// Currently no way to autocomplete regular expressions.
	return false
}

//func (g *Grep) execute(cos commands.CommandOS, args, flags map[string]*commands.Value, oi *commands.OptionInfo) (*commands.ExecutorResponse, bool) {
func (g *Grep) Execute(ws *commands.WorldState) bool {
	ignoreCase := ws.Flags[caseFlagName].Bool()

	var ffs filterFuncs //[]func(string) (*formatter, bool)
	for _, pattern := range ws.Args[patternArgName].StringList() {
		if ignoreCase {
			pattern = fmt.Sprintf("(?i)%s", pattern)
		}
		r, err := regexp.Compile(pattern)
		if err != nil {
			ws.Cos.Stderr("invalid regex: %v", err)
			return false
		}
		ffs = append(ffs, colorMatch(r))
	}

	for _, pattern := range ws.Flags[invertFlagName].StringList() {
		r, err := regexp.Compile(pattern)
		if err != nil {
			ws.Cos.Stderr("invalid invert regex: %v", err)
			return false
		}
		ffs = append(ffs, func(s string) (*formatter, bool) { return nil, !r.MatchString(s) })
	}

	return g.inputSource.Process(ws, ffs)
}

func (g *Grep) Node() *commands.Node {
	flags := append(g.inputSource.Flags(), caseFlag, invertFlag)
	flagNode := commands.NewFlagNode(flags...)
	return commands.SerialNodes(flagNode, patternArg, g)
}

/*func (g *Grep) Node() commands.NodeProcessor {
	flags := append(g.inputSource.Flags(), caseFlag, invertFlag)
	return &commands.Node{
		Processor: commands.ExecutorNode(g.),
	}
}*/

/*func (g *Grep) Command() commands.Command {
	flags := []commands.Flag{
		caseFlag,
		invertFlag,
	}
	return &commands.CommandBranch{
		Subcommands: g.inputSource.Subcommands(),
		TerminusCommand: &commands.TerminusCommand{
			Executor: g.execute,
			Args: []commands.Arg{
				patternArg,
			},
			Flags: append(flags, g.inputSource.Flags()...),
		},
	}
}*/
