package cli

import (
	"bytes"
	"flag"
	"reflect"
	"strings"
	"testing"
)

func testCli(env map[string]string) (*Cli, *bytes.Buffer, *bytes.Buffer) {
	var stdout, stderr bytes.Buffer
	c := NewCli()
	c.stdout = &stdout
	c.stderr = &stderr
	c.lookupEnv = func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}
	return c, &stdout, &stderr
}

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		wantOut string // substring of stdout
		wantErr string // substring of the error; "" means no error
	}{
		{name: "no args prints help", args: nil, wantOut: "Commands:"},
		{name: "-h", args: []string{"-h"}, wantOut: "Commands:"},
		{name: "help command", args: []string{"help", "template", "generic"}, wantOut: "--csv-separator"},
		{name: "subcommand -h", args: []string{"template", "generic", "--help"}, wantOut: "-o, --output string"},
		{name: "version", args: []string{"version"}, wantOut: appName + " development"},
		{name: "global flag before command", args: []string{"-q", "--log-level", "debug", "version"}, wantOut: "development"},
		{name: "global flag after command", args: []string{"version", "--log-level=error"}, wantOut: "development"},
		{name: "unknown command", args: []string{"nope"}, wantErr: `unknown command "nope"`},
		{name: "unknown subcommand", args: []string{"template", "nope"}, wantErr: `unknown command "nope" for "iso20022tpl template"`},
		{name: "unknown flag", args: []string{"version", "--nope"}, wantErr: "flag provided but not defined"},
		{name: "missing arg", args: []string{"json"}, wantErr: "accepts 1 arg(s), received 0"},
		{name: "extra arg", args: []string{"version", "x"}, wantErr: "accepts 0 arg(s), received 1"},
		{name: "generate requires flags", args: []string{"generate"}, wantErr: "missing required flags"},
		{name: "invalid log level", args: []string{"--log-level", "loud", "version"}, wantErr: "invalid log level"},
		{name: "env sets flag", args: []string{"version"}, env: map[string]string{"ISO20022TPL_LOG_LEVEL": "loud"}, wantErr: "invalid log level"},
		{name: "flag overrides env", args: []string{"--log-level", "info", "version"}, env: map[string]string{"ISO20022TPL_LOG_LEVEL": "loud"}, wantOut: "development"},
		{name: "invalid bool env", args: []string{"version"}, env: map[string]string{"ISO20022TPL_QUIET": "maybe"}, wantErr: "ISO20022TPL_QUIET"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, stdout, _ := testCli(tt.env)
			err := c.Run(tt.args)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
			if !strings.Contains(stdout.String(), tt.wantOut) {
				t.Errorf("stdout = %q, want substring %q", stdout.String(), tt.wantOut)
			}
		})
	}
}

func TestEnvAndShortFlags(t *testing.T) {
	c, _, _ := testCli(map[string]string{"ISO20022TPL_QUIET": "true", "ISO20022TPL_JSON_LOG": "1"})
	if err := c.Run([]string{"version"}); err != nil {
		t.Fatal(err)
	}
	if !c.quiet || !c.jsonLog {
		t.Errorf("quiet=%v jsonLog=%v, want both true", c.quiet, c.jsonLog)
	}

	// A short flag on the command line wins over the env var of its long name.
	c, _, _ = testCli(map[string]string{"ISO20022TPL_QUIET": "true"})
	if err := c.Run([]string{"version", "-q=false"}); err != nil {
		t.Fatal(err)
	}
	if c.quiet {
		t.Error("quiet = true, want false")
	}
}

func TestParseInterspersed(t *testing.T) {
	tests := []struct {
		args    []string
		wantPos []string
		wantOut string
	}{
		{[]string{"a", "b"}, nil, ""},
		{[]string{"-o", "x", "a", "b"}, nil, "x"},
		{[]string{"a", "-o", "x", "b"}, nil, "x"},
		{[]string{"a", "b", "--output=x"}, nil, "x"},
		{[]string{"a", "--", "-o", "b"}, []string{"a", "-o", "b"}, ""},
	}
	for _, tt := range tests {
		fs := newFlagSet("t")
		var out string
		fs.string(&out, "output", "o", "", "")
		pos, err := parseInterspersed(fs.FlagSet, tt.args)
		if err != nil {
			t.Fatalf("%v: %v", tt.args, err)
		}
		want := tt.wantPos
		if want == nil {
			want = []string{"a", "b"}
		}
		if !reflect.DeepEqual(pos, want) || out != tt.wantOut {
			t.Errorf("%v: pos=%v out=%q, want pos=%v out=%q", tt.args, pos, out, want, tt.wantOut)
		}
	}
}

func TestHelpReturnsNil(t *testing.T) {
	c, _, _ := testCli(nil)
	if err := c.Run([]string{"json", "-h"}); err != nil {
		t.Fatalf("got %v, want nil (not %v)", err, flag.ErrHelp)
	}
}
