/*
Copyright © 2021 Michael Gruener

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"
)

// Cli is the core command line interface.
type Cli struct {
	stdout    io.Writer
	stderr    io.Writer
	lookupEnv func(string) (string, bool)

	logLevel     string
	jsonLog      bool
	logFunctions bool
	quiet        bool
}

// NewCli creates a Cli that writes to os.Stdout and os.Stderr and reads
// ISO20022TPL_* environment variables.
func NewCli() *Cli {
	return &Cli{
		stdout:    os.Stdout,
		stderr:    os.Stderr,
		lookupEnv: os.LookupEnv,
		logLevel:  "warn",
	}
}

// command is a node in the command tree. A command either has
// subcommands or a run function.
type command struct {
	name  string
	args  string // positional arguments shown in the usage line
	short string
	nargs int // exact number of positional arguments
	flags func(fs *flagSet)
	run   func(c *Cli, args []string) error
	sub   []*command
}

func rootCommand() *command {
	return &command{
		name:  appName,
		short: "Convert ISO 20022 bank statements to CSV",
		sub: []*command{
			newVersionCmd(),
			newJsonCmd(),
			newYamlCmd(),
			newTemplateCmd(),
			newCrawlCmd(),
			newGenerateCmd(),
		},
	}
}

// Run parses args (without the program name) and executes the selected
// command.
func (c *Cli) Run(args []string) error {
	err := c.run(rootCommand(), []string{appName}, args, map[string]bool{})
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	return err
}

func (c *Cli) run(cmd *command, path, args []string, set map[string]bool) error {
	fs := c.flagSet(cmd, path)

	if len(cmd.sub) > 0 {
		if err := fs.Parse(args); err != nil {
			return c.flagError(err, cmd, path, fs)
		}
		fs.markSet(set)
		rest := fs.Args()
		if len(rest) == 0 {
			c.usage(c.stdout, cmd, path, fs)
			return nil
		}
		if rest[0] == "help" {
			return c.help(cmd, path, rest[1:])
		}
		if sub := cmd.find(rest[0]); sub != nil {
			return c.run(sub, append(path, sub.name), rest[1:], set)
		}
		c.usage(c.stderr, cmd, path, fs)
		return fmt.Errorf("unknown command %q for %q", rest[0], strings.Join(path, " "))
	}

	rest, err := parseInterspersed(fs.FlagSet, args)
	if err != nil {
		return c.flagError(err, cmd, path, fs)
	}
	fs.markSet(set)
	if err := fs.applyEnv(set, c.lookupEnv); err != nil {
		return err
	}
	if len(rest) != cmd.nargs {
		return fmt.Errorf("%q accepts %d arg(s), received %d\nusage: %s",
			strings.Join(path, " "), cmd.nargs, len(rest), usageLine(cmd, path))
	}
	if err := c.setupLogger(); err != nil {
		return err
	}
	return cmd.run(c, rest)
}

// help prints the usage of the command found by following names from cmd.
func (c *Cli) help(cmd *command, path, names []string) error {
	for _, name := range names {
		sub := cmd.find(name)
		if sub == nil {
			return fmt.Errorf("unknown command %q for %q", name, strings.Join(path, " "))
		}
		cmd, path = sub, append(path, sub.name)
	}
	c.usage(c.stdout, cmd, path, c.flagSet(cmd, path))
	return nil
}

func (c *Cli) flagError(err error, cmd *command, path []string, fs *flagSet) error {
	if errors.Is(err, flag.ErrHelp) {
		c.usage(c.stdout, cmd, path, fs)
		return err
	}
	c.usage(c.stderr, cmd, path, fs)
	return err
}

func (cmd *command) find(name string) *command {
	for _, s := range cmd.sub {
		if s.name == name {
			return s
		}
	}
	return nil
}

// flagSet returns the flags accepted by cmd: the global flags plus its own.
// Global flags are registered with their current values as defaults, so
// values parsed at an earlier level are kept.
func (c *Cli) flagSet(cmd *command, path []string) *flagSet {
	fs := newFlagSet(strings.Join(path, " "))
	fs.string(&c.logLevel, "log-level", "", c.logLevel, "log level (debug,info,warn,error)")
	fs.bool(&c.jsonLog, "json-log", "", c.jsonLog, "log as json")
	fs.bool(&c.logFunctions, "log-functions", "", c.logFunctions, "include the source file and line in log messages")
	fs.bool(&c.quiet, "quiet", "q", c.quiet, "suppress progress messages")
	if cmd.flags != nil {
		cmd.flags(fs)
	}
	return fs
}

func usageLine(cmd *command, path []string) string {
	line := strings.Join(path, " ")
	if len(cmd.sub) > 0 {
		return line + " [flags] <command>"
	}
	if cmd.args != "" {
		line += " " + cmd.args
	}
	return line + " [flags]"
}

func (c *Cli) usage(w io.Writer, cmd *command, path []string, fs *flagSet) {
	fmt.Fprintf(w, "%s\n\nUsage:\n  %s\n", cmd.short, usageLine(cmd, path))
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	if len(cmd.sub) > 0 {
		fmt.Fprintln(tw, "\nCommands:")
		for _, s := range cmd.sub {
			fmt.Fprintf(tw, "  %s\t%s\n", s.name, s.short)
		}
		fmt.Fprintf(tw, "  help\tHelp about any command\n")
	}
	fmt.Fprintln(tw, "\nFlags:")
	fs.printDefaults(tw)
	_ = tw.Flush()
}

func (c *Cli) setupLogger() error {
	// --quiet hides progress messages but keeps errors and command results.
	if c.quiet {
		progress = io.Discard
	} else {
		progress = c.stdout
	}

	level, err := parseLevel(c.logLevel)
	if err != nil {
		return err
	}
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: c.logFunctions,
	}
	var h slog.Handler = slog.NewTextHandler(c.stderr, opts)
	if c.jsonLog {
		h = slog.NewJSONHandler(c.stderr, opts)
	}
	slog.SetDefault(slog.New(h))
	return nil
}

// parseLevel maps a --log-level value to a slog level. The logrus level
// names used by earlier versions (trace, warning, fatal, panic) are still
// accepted.
func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "trace":
		return slog.LevelDebug, nil
	case "warning":
		return slog.LevelWarn, nil
	case "fatal", "panic":
		return slog.LevelError, nil
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return 0, fmt.Errorf("invalid log level %q: use debug, info, warn or error", s)
	}
	return l, nil
}
