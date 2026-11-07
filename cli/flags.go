package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

const envPrefix = "ISO20022TPL_"

// flagSet is a flag.FlagSet whose flags can have a one-letter alias.
type flagSet struct {
	*flag.FlagSet
	shortOf map[string]string // long name -> short name
	longOf  map[string]string // short name -> long name
}

func newFlagSet(name string) *flagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	// Errors and help are printed by the caller.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	return &flagSet{
		FlagSet: fs,
		shortOf: map[string]string{},
		longOf:  map[string]string{},
	}
}

func (fs *flagSet) string(p *string, long, short, def, usage string) {
	fs.StringVar(p, long, def, usage)
	if short != "" {
		fs.StringVar(p, short, def, usage)
		fs.alias(long, short)
	}
}

func (fs *flagSet) bool(p *bool, long, short string, def bool, usage string) {
	fs.BoolVar(p, long, def, usage)
	if short != "" {
		fs.BoolVar(p, short, def, usage)
		fs.alias(long, short)
	}
}

func (fs *flagSet) alias(long, short string) {
	fs.shortOf[long] = short
	fs.longOf[short] = long
}

// markSet records the long names of all flags set on the command line.
func (fs *flagSet) markSet(set map[string]bool) {
	fs.Visit(func(f *flag.Flag) {
		name := f.Name
		if long, ok := fs.longOf[name]; ok {
			name = long
		}
		set[name] = true
	})
}

// applyEnv sets every flag not set on the command line from its
// environment variable, e.g. --log-level from ISO20022TPL_LOG_LEVEL.
func (fs *flagSet) applyEnv(set map[string]bool, lookup func(string) (string, bool)) error {
	var err error
	fs.VisitAll(func(f *flag.Flag) {
		if err != nil || set[f.Name] {
			return
		}
		if _, isShort := fs.longOf[f.Name]; isShort {
			return
		}
		key := envPrefix + strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		v, ok := lookup(key)
		if !ok {
			return
		}
		if e := fs.Set(f.Name, v); e != nil {
			err = fmt.Errorf("invalid value %q for %s: %w", v, key, e)
		}
	})
	return err
}

// printDefaults writes one line per flag, merging aliases.
func (fs *flagSet) printDefaults(w io.Writer) {
	fs.VisitAll(func(f *flag.Flag) {
		if _, isShort := fs.longOf[f.Name]; isShort {
			return
		}
		name := "    --" + f.Name
		if short, ok := fs.shortOf[f.Name]; ok {
			name = "-" + short + ", --" + f.Name
		}
		typ, usage := flag.UnquoteUsage(f)
		if typ != "" {
			name += " " + typ
		}
		if f.DefValue != "" && f.DefValue != "false" {
			usage += fmt.Sprintf(" (default %q)", f.DefValue)
		}
		fmt.Fprintf(w, "  %s\t%s\n", name, usage)
	})
}

// parseInterspersed parses flags that may appear before, between or after
// positional arguments and returns the positional arguments. Everything
// after "--" is positional.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		consumed := len(args) - len(rest)
		if consumed > 0 && args[consumed-1] == "--" {
			return append(pos, rest...), nil
		}
		if len(rest) == 0 {
			return pos, nil
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}
