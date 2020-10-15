package grep

import (
	"fmt"
	"regexp"

	"github.com/leep-frog/commands/color"
	"github.com/leep-frog/commands/commands"
)

var (
	patternArg = commands.StringListArg("pattern", 0, -1, nil)
	caseFlag   = commands.BoolFlag("ignoreCase", 'i')
	invertFlag = commands.StringListFlag("invert", 'v', 0, -1, nil)
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
	Process(cos commands.CommandOS, args, flags map[string]*commands.Value, oi *commands.OptionInfo, filters filterFuncs) (*commands.ExecutorResponse, bool)
	Flags() []commands.Flag
	Option() *commands.Option
}

type Grep struct {
	caseSensitive bool
	inputSource   inputSource
}

func (*Grep) Load(jsn string) error { return nil }
func (*Grep) Changed() bool         { return false }

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

func (g *Grep) execute(cos commands.CommandOS, args, flags map[string]*commands.Value, oi *commands.OptionInfo) (*commands.ExecutorResponse, bool) {
	ignoreCase := flags[caseFlag.Name()].Bool() != nil && *flags[caseFlag.Name()].Bool()

	var ffs filterFuncs //[]func(string) (*formatter, bool)
	if patterns := args[patternArg.Name()].StringList(); patterns != nil {
		for _, pattern := range *patterns {
			if ignoreCase {
				pattern = fmt.Sprintf("(?i)%s", pattern)
			}
			r, err := regexp.Compile(pattern)
			if err != nil {
				cos.Stderr("invalid regex: %v", err)
				return nil, false
			}
			// TODO: color response (regexp.FindStringIndex)
			ffs = append(ffs, colorMatch(r))
		}
	}

	if inverts := flags[invertFlag.Name()].StringList(); inverts != nil {
		for _, pattern := range *inverts {
			r, err := regexp.Compile(pattern)
			if err != nil {
				cos.Stderr("invalid invert regex: %v", err)
				return nil, false
			}
			ffs = append(ffs, func(s string) (*formatter, bool) { return nil, !r.MatchString(s) })
		}
	}

	return g.inputSource.Process(cos, args, flags, oi, ffs)
}

func (g *Grep) Command() commands.Command {
	flags := []commands.Flag{
		caseFlag,
		invertFlag,
	}
	return &commands.TerminusCommand{
		Executor: g.execute,
		Args: []commands.Arg{
			patternArg,
		},
		Flags: append(flags, g.inputSource.Flags()...),
	}
}
