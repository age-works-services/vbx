package commands

import (
	"context"
	"fmt"
	"vbx/internal/command"
	"vbx/internal/project"
)

func init() { register(newNameCommand) }
func newNameCommand(d Dependencies) command.Command {
	out := d.Out
	return command.Handler{Definition: command.Spec{Name: "name", Summary: "Print the canonical project VM name", Usage: "name"}, Execute: func(ctx context.Context, in command.Invocation) error {
		p, err := project.Resolve(in.General.Project)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, p.Name)
		return err
	}}
}
