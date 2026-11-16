package commands

import (
	"context"
	"fmt"
	"vbx/internal/command"
	"vbx/internal/project"
)

func init() { register(newStatusCommand) }
func newStatusCommand(d Dependencies) command.Command {
	inventory, out := d.Inventory, d.Out
	return command.Handler{Definition: command.Spec{Name: "status", Summary: "Show the project VM's current status", Usage: "status"}, Execute: func(ctx context.Context, in command.Invocation) error {
		p, err := project.Resolve(in.General.Project)
		if err != nil {
			return err
		}
		instances, err := inventory.List(ctx)
		if err != nil {
			return err
		}
		status := "Missing"
		for _, instance := range instances {
			if instance.Name == p.Name {
				status = instance.Status
				break
			}
		}
		_, err = fmt.Fprintf(out, "%s: %s\n", p.Name, status)
		return err
	}}
}
