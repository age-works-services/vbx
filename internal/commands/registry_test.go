package commands

import (
	"context"
	"io"
	"strings"
	"testing"
	"vbx/internal/cli"
	"vbx/internal/command"
)

func TestCommandNamesInHelp(t *testing.T) {
	cmds := New(Dependencies{})
	var root strings.Builder
	if err := cli.Run(context.Background(), []string{"--help"}, cmds, &root); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"check-host-prerequisites", "show-host-ssh-config-line", "list-vms", "show-project-effective-setup"} {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(root.String(), name) {
				t.Fatal(root.String())
			}
			var out strings.Builder
			if err := cli.Run(context.Background(), []string{name, "--help"}, cmds, &out); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "Usage: vbx "+name+"\n") {
				t.Fatal(out.String())
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	d := Dependencies{}
	commands := New(d)
	if len(commands) == 0 {
		t.Fatal("empty registry")
	}
	extra := func(Dependencies) command.Command {
		return command.Handler{Definition: command.Spec{Name: "xyz", Summary: "Extra command", Usage: "xyz"}, Execute: func(context.Context, command.Invocation) error { return nil }}
	}
	isolated := append(append([]Factory{}, factories...), extra)
	cmds := NewWith(d, isolated)
	var out strings.Builder
	if err := cli.Run(context.Background(), []string{"--help"}, cmds, &out); err != nil || !strings.Contains(out.String(), "xyz") {
		t.Fatal(out.String(), err)
	}
	if err := cli.Run(context.Background(), []string{"xyz"}, cmds, io.Discard); err != nil {
		t.Fatal(err)
	}
	if len(factories) != len(commands) {
		t.Fatal("global registry polluted")
	}
	if err := cli.Validate(NewWith(d, append(isolated, extra))); err == nil {
		t.Fatal("duplicate accepted")
	}
	if err := cli.Run(context.Background(), []string{"start", "--terminal", "--vscode"}, commands, io.Discard); err == nil {
		t.Fatal("conflicting targets accepted")
	}
}
