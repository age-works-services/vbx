package commands

import (
	"context"
	"fmt"
	"strings"
	"vbx/internal/command"
)

func init() { register(newShowHostSSHConfigLineCommand) }
func newShowHostSSHConfigLineCommand(d Dependencies) command.Command {
	paths, out := d.Paths, d.Out
	return command.Handler{Definition: command.Spec{Name: "show-host-ssh-config-line", Summary: "Print the SSH Include line", Usage: "show-host-ssh-config-line"}, Execute: func(ctx context.Context, in command.Invocation) error {
		dir, err := paths.SSHDir()
		if err != nil {
			return err
		}
		escaped := strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(dir)
		_, err = fmt.Fprintf(out, "Include \"%s/*.conf\"\n", escaped)
		return err
	}}
}
