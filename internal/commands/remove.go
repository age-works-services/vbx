package commands

import (
	"context"
	"fmt"
	"os"
	"vbx/internal/command"
)

func init() { register(newRemoveCommand) }
func newRemoveCommand(d Dependencies) command.Command {
	scopes, backend, prompt := d.Scopes, d.Lima, d.Interaction
	return command.Handler{Definition: command.Spec{Name: "remove", Summary: "Confirm and delete the VM and guest-local files", Usage: "remove"}, Execute: func(ctx context.Context, in command.Invocation) error {
		scope, err := scopes.Acquire(ctx, in.General.Project)
		if err != nil {
			return err
		}
		defer scope.Release()
		if !scope.VMExists && !scope.StateExists {
			return fmt.Errorf("project VM does not exist")
		}
		name := scope.Project.Name
		answer, err := prompt.ReadLine(ctx, fmt.Sprintf("Delete %s and all guest-local files? Type the VM name: ", name))
		if err != nil {
			return err
		}
		if answer != name {
			return fmt.Errorf("removal cancelled")
		}
		if scope.VMExists {
			status, err := backend.Status(ctx, name)
			if err != nil {
				return err
			}
			switch status {
			case "Running":
				if err = backend.Stop(ctx, name); err != nil {
					return err
				}
			case "Stopped":
			default:
				return fmt.Errorf("cannot safely remove VM with status %s", status)
			}
			if err = backend.Delete(ctx, name); err != nil {
				return err
			}
		}
		if err = os.RemoveAll(scope.InstanceDir); err != nil {
			return err
		}
		if err = os.Remove(scope.SSHLink); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}}
}
