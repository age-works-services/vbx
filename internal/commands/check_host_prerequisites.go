package commands

import (
	"context"
	"fmt"
	"vbx/internal/command"
)

func init() { register(newCheckHostPrerequisitesCommand) }
func newCheckHostPrerequisitesCommand(d Dependencies) command.Command {
	host, runner, out, errOut := d.Host, d.Runner, d.Out, d.Err
	return command.Handler{Definition: command.Spec{Name: "check-host-prerequisites", Summary: "Check Linux host prerequisites", Usage: "check-host-prerequisites"}, Execute: func(ctx context.Context, in command.Invocation) error {
		if err := host.Check(ctx); err != nil {
			return err
		}
		for _, tool := range []struct {
			name string
			args []string
		}{{"limactl", []string{"--version"}}, {"qemu-system-x86_64", []string{"--version"}}, {"qemu-img", []string{"--version"}}, {"ssh", []string{"-V"}}} {
			if err := runner.Run(ctx, nil, out, errOut, tool.name, tool.args...); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintln(out, "Host prerequisites passed.")
		return err
	}}
}
