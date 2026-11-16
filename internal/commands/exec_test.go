package commands

import (
	"context"
	"errors"
	"io"
	"testing"
	"vbx/internal/cli"
	"vbx/internal/command"
)

func TestExecRequiresCommand(t *testing.T) {
	cmds := New(Dependencies{})
	var err error
	for _, args := range [][]string{{"exec"}, {"exec", "--"}} {
		err = cli.Run(context.Background(), args, cmds, io.Discard)
		var usage *command.UsageError
		if !errors.As(err, &usage) || err.Error() != "exec requires a guest command" {
			t.Fatal(err)
		}
	}
}
