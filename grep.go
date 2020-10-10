package grep

import (
	"regexp"

	"github.com/leep-frog/cli/commands"
)

var (
	patternArg = commands.StringListArg("pattern", 0, -1, nil)
	caseFlag   = commands.NewBooleanFlag("caseInsensitve", 'i', false)
	invertFlag = commands.StringListFlag("invert", 'v', 0, -1, nil)
	// TODO: or pattern
)

type filterFunc func(string) bool

type inputSource interface {
	Name() string
	Alias() string
	Process(cos commands.CommandOS, args, flags map[string]*commands.Value, filter filterFunc) (*commands.ExecutorResponse, bool)
	Flags() []commands.Flag
}

type grep struct {
	caseSensitive bool
	inputSource   inputSource
}

func (*grep) Load(jsn string) error { return nil }
func (*grep) Changed() bool         { return false }

func (g *grep) Name() string {
	return g.inputSource.Name()
}

func (g *grep) Alias() string {
	return g.inputSource.Alias()
}

func (g *grep) execute(cos commands.CommandOS, args, flags map[string]*commands.Value) (*commands.ExecutorResponse, bool) {
	var filterFuncs []func(string) bool

	// TODO: case flag, boolean flag
	//toLower := flags[caseFlag.Name()].Bool()

	if patterns := args[patternArg.Name()].StringList(); patterns != nil {
		for _, pattern := range *patterns {
			r, err := regexp.Compile(pattern)
			if err != nil {
				cos.Stderr("invalid regex: %v", err)
				return nil, false
			}
			filterFuncs = append(filterFuncs, r.MatchString)
		}
	}

	if inverts := flags[invertFlag.Name()].StringList(); inverts != nil {
		for _, pattern := range *inverts {
			r, err := regexp.Compile(pattern)
			if err != nil {
				cos.Stderr("invalid invert regex: %v", err)
				return nil, false
			}
			filterFuncs = append(filterFuncs, func(s string) bool { return !r.MatchString(s) })
		}
	}

	filterFunc := func(s string) bool {
		for _, ff := range filterFuncs {
			if !ff(s) {
				return false
			}
		}
		return true
	}

	return g.inputSource.Process(cos, args, flags, filterFunc)
}

func (g *grep) Command() commands.Command {
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
