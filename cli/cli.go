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
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Cli is the core command line interface
// Wrapping the cli in its own public type might allow
// it to be integrated in other programs, creating
// something like a composite application
type Cli struct {
	RootCommand *cobra.Command
	viper       *viper.Viper
}

// NewCli creates a
func NewCli() *Cli {
	v := viper.GetViper()
	v.SetEnvPrefix(appName)
	dashReplacer := strings.NewReplacer("-", "_", ".", "_")
	v.SetEnvKeyReplacer(dashReplacer)
	v.AutomaticEnv()
	cli := &Cli{
		viper: v,
	}
	cli.RootCommand = NewRootCommand(cli)
	return cli
}

func (c *Cli) bindAllFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if err := c.viper.BindPFlag(f.Name, cmd.PersistentFlags().Lookup(f.Name)); err != nil {
			panic(err) // Should never happen
		}
	})
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if err := c.viper.BindPFlag(f.Name, cmd.Flags().Lookup(f.Name)); err != nil {
			panic(err) // Should never happen
		}
	})
}

// wrapper func to bind all flags with viper on command execution
// and to perform global post-command execution steps (if necessary)
func (c *Cli) wrap(f func(*Cli, *cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		if err := c.setupLogger(); err != nil {
			return err
		}
		c.bindAllFlags(cmd)
		return f(c, cmd, args)
	}
}

func (c *Cli) setupLogger() error {
	// --quiet hides progress messages but keeps errors and command results.
	if c.viper.GetBool("quiet") {
		progress = io.Discard
	} else {
		progress = os.Stdout
	}

	level, err := parseLevel(c.viper.GetString("log-level"))
	if err != nil {
		return err
	}
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: c.viper.GetBool("log-functions"),
	}
	var h slog.Handler = slog.NewTextHandler(os.Stderr, opts)
	if c.viper.GetBool("json-log") {
		h = slog.NewJSONHandler(os.Stderr, opts)
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

func NewRootCommand(c *Cli) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:           "iso20022tpl",
		Short:         "Convert ISO 20022 bank statements to CSV",
		SilenceUsage:  false,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().String("log-level", "warn", "log level (debug,info,warn,error)")
	rootCmd.PersistentFlags().Bool("json-log", false, "log as json")
	rootCmd.PersistentFlags().Bool("log-functions", false, "include the source file and line in log messages")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress all normal output")

	c.bindAllFlags(rootCmd)

	rootCmd.AddCommand(
		newVersionCmd(),
		newJsonCmd(c),
		newYamlCmd(c),
		newTemplateCmd(c),
		newCrawlCmd(c),
		newGenerateCmd(c),
	)

	return rootCmd
}
