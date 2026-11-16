package commands

import (
	"context"
	"fmt"
	"vbx/internal/command"
)

func init() { register(newStopCommand) }
func newStopCommand(d Dependencies) command.Command {
	scopes := d.Scopes
	var backend stopper = d.Lima
	return command.Handler{Definition: command.Spec{Name: "stop", Summary: "Stop an existing project VM", Usage: "stop"}, Execute: func(ctx context.Context, in command.Invocation) error {
		scope, err := scopes.Acquire(ctx, in.General.Project)
		if err != nil {
			return err
		}
		defer scope.Release()
		if !scope.VMExists {
			return fmt.Errorf("project VM does not exist")
		}
		return backend.Stop(ctx, scope.Project.Name)
	}}
}
