package cli

import (
	"context"
	"io"
	"reflect"
	"testing"
	"vbx/internal/command"
)

func TestParsing(t *testing.T) {
	for _, tt := range []struct {
		args             []string
		command, project string
		argv             []string
		bad, help        bool
	}{
		{args: []string{}, command: "start", project: "."},
		{args: []string{"--project", "X", "start", "--vscode"}, command: "start", project: "X"},
		{args: []string{"start", "--vscode", "--project=X"}, command: "start", project: "X"},
		{args: []string{"--project", "start", "stop"}, command: "stop", project: "start"},
		{args: []string{"exec", "--project", "X", "npm", "--project", "other"}, command: "exec", project: "X", argv: []string{"npm", "--project", "other"}},
		{args: []string{"exec", "--", "--tool", "a b; $(echo hi)"}, command: "exec", project: ".", argv: []string{"--tool", "a b; $(echo hi)"}},
		{args: []string{"--help"}, help: true}, {args: []string{"--help", "start"}, help: true}, {args: []string{"exec", "--help"}, help: true},
		{args: []string{"--project", "X", "start", "--project", "Y"}, bad: true},
		{args: []string{"start", "--project", "--vscode"}, bad: true}, {args: []string{"start", "--vscode=false"}, bad: true}, {args: []string{"-project", "X", "start"}, bad: true},
		{args: []string{"--", "start"}, bad: true}, {args: []string{"--vscode", "start"}, bad: true}, {args: []string{"start", "stop"}, bad: true}, {args: []string{"bogus"}, bad: true},
	} {

		t.Run(join(tt.args), func(t *testing.T) {
			called := ""
			var got command.GeneralOptions
			var argv []string
			cmds := []command.Command{}
			for _, name := range []string{"start", "stop", "exec"} {
				n := name
				policy := command.NoOperands
				if n == "exec" {
					policy = command.OptionalOperands
				}
				cmds = append(cmds, command.Handler{Definition: command.Spec{Name: n, Summary: n, Usage: n, Operands: policy, Flags: map[string]command.Flag{"vscode": {Summary: "launch"}}}, Execute: func(ctx context.Context, in command.Invocation) error {
					called = n
					got = in.General
					argv = in.Args
					return nil
				}})
			}
			err := Run(context.Background(), tt.args, cmds, io.Discard)
			if (err != nil) != tt.bad {
				t.Fatalf("err=%v", err)
			}
			if tt.bad || tt.help {
				if called != "" {
					t.Fatal("handler ran")
				}
				return
			}
			if called != tt.command || got.Project != tt.project || !reflect.DeepEqual(argv, tt.argv) {
				t.Fatalf("got %s %#v %#v", called, got, argv)
			}
		})
	}
}
func join(a []string) string {
	out := ""
	for _, s := range a {
		out += s + " "
	}
	return out
}
