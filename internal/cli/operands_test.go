package cli

import (
	"context"
	"io"
	"reflect"
	"testing"
	"vbx/internal/command"
)

func TestOptionalOperands(t *testing.T) {
	for _, args := range [][]string{{"optional"}, {"optional", "one", "--project", "guest"}, {"optional", "--project", "host", "--", "--guest"}} {
		t.Run(join(args), func(t *testing.T) {
			var got []string
			called := false
			commands := []command.Command{
				command.Handler{Definition: command.Spec{Name: "start", Summary: "default", Usage: "start"}, Execute: func(context.Context, command.Invocation) error { return nil }},
				command.Handler{Definition: command.Spec{Name: "optional", Summary: "Optional operands", Usage: "optional [ARGS...]", Operands: command.OptionalOperands}, Execute: func(ctx context.Context, in command.Invocation) error { called = true; got = in.Args; return nil }},
			}
			if err := Run(context.Background(), args, commands, io.Discard); err != nil {
				t.Fatal(err)
			}
			var want []string
			if len(args) > 1 {
				want = args[1:]
				if args[1] == "--project" {
					want = []string{"--guest"}
				}
			}
			if !called || !reflect.DeepEqual(got, want) {
				t.Fatal(called, got, want)
			}
		})
	}
}
