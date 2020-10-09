package grep

import (
	"fmt"
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
	Process(args, flags map[string]*commands.Value, filter filterFunc) ([]string, error)
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

func (g *grep) execute(args, flags map[string]*commands.Value) (*commands.ExecutorResponse, error) {
	var filterFuncs []func(string) bool

	// TODO: case flag, boolean flag
	//toLower := flags[caseFlag.Name()].Bool()

	if patterns := args[patternArg.Name()].StringList(); patterns != nil {
		for _, pattern := range *patterns {
			r, err := regexp.Compile(pattern)
			if err != nil {
				return &commands.ExecutorResponse{
					Stderr: []string{fmt.Sprintf("invalid regex: %v", err)},
				}, nil
			}
			filterFuncs = append(filterFuncs, r.MatchString)
		}
	}

	if inverts := flags[invertFlag.Name()].StringList(); inverts != nil {
		for _, pattern := range *inverts {
			r, err := regexp.Compile(pattern)
			if err != nil {
				return &commands.ExecutorResponse{
					Stderr: []string{fmt.Sprintf("invalid invert regex: %v", err)},
				}, nil
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

	results, err := g.inputSource.Process(args, flags, filterFunc)
	if err != nil {
		return &commands.ExecutorResponse{
			Stderr: []string{err.Error()},
		}, nil
	}
	return &commands.ExecutorResponse{
		Stdout: results,
	}, nil
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
