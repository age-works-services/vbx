package cli

import (
	"context"
	"io"
	"strings"
	"testing"
	"vbx/internal/command"
)

func synthetic(spec command.Spec) command.Command {
	return command.Handler{Definition: spec, Execute: func(context.Context, command.Invocation) error { panic("unexpected execution") }}
}
func TestMetadataValidation(t *testing.T) {
	start := command.Spec{Name: "start", Summary: "start", Usage: "start"}
	for _, flag := range []string{"project", "help", "h", "-bad", "has space", "has=value", ""} {
		t.Run(flag, func(t *testing.T) {
			s := start
			s.Flags = map[string]command.Flag{flag: {Summary: "bad"}}
			if err := Validate([]command.Command{synthetic(s)}); err == nil {
				t.Fatal("invalid flag accepted")
			}
		})
	}
	s := start
	s.Flags = map[string]command.Flag{"good": {Kind: command.FlagKind(99)}}
	if err := Validate([]command.Command{synthetic(s)}); err == nil {
		t.Fatal("unsupported kind accepted")
	}
	if err := Validate([]command.Command{synthetic(start), synthetic(start)}); err == nil {
		t.Fatal("duplicate accepted")
	}
	if err := Validate(nil); err == nil {
		t.Fatal("missing default accepted")
	}
}
func TestHelpBeforeOperandAndHandlerValidation(t *testing.T) {
	commands := []command.Command{
		synthetic(command.Spec{Name: "start", Summary: "start", Usage: "start"}),
		synthetic(command.Spec{Name: "required", Summary: "required", Usage: "required ARG", Operands: command.RequiredOperands}),
	}
	for _, args := range [][]string{{"required", "--help"}, {"start", "--help", "unexpected"}, {"--help", "--help"}} {
		var out strings.Builder
		if err := Run(context.Background(), args, commands, &out); err != nil || !strings.Contains(out.String(), "Usage:") {
			t.Fatal(args, err, out.String())
		}
	}
	if err := Run(context.Background(), []string{"start", "--help", "--unknown"}, commands, io.Discard); err == nil {
		t.Fatal("help hid syntax error")
	}
}
