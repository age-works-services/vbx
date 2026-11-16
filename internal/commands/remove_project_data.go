package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"vbx/internal/command"
	"vbx/internal/project"
	"vbx/internal/state"
)

func init() { register(newRemoveProjectDataCommand) }
func newRemoveProjectDataCommand(d Dependencies) command.Command {
	paths, inventory, prompt, out := d.Paths, d.Inventory, d.Interaction, d.Out
	return command.Handler{Definition: command.Spec{Name: "remove-project-data", Summary: "Confirm and delete retained project data", Usage: "remove-project-data [PROJECT-ID]", Operands: command.OptionalOperands}, Execute: func(ctx context.Context, in command.Invocation) error {
		if len(in.Args) > 1 {
			return &command.UsageError{Message: "remove-project-data accepts at most one project identifier"}
		}
		var id string
		if len(in.Args) == 1 {
			id = in.Args[0]
			if !state.ValidProjectID(id) {
				return &command.UsageError{Message: "invalid project identifier"}
			}
		} else {
			p, err := project.Resolve(in.General.Project)
			if err != nil {
				return err
			}
			id = p.Name
		}
		base, err := paths.StateDir()
		if err != nil {
			return err
		}
		unlock, err := state.Lock(base, id)
		if err != nil {
			return err
		}
		defer unlock()
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		instances, err := inventory.List(ctx)
		if err != nil {
			return err
		}
		for _, vm := range instances {
			if vm.Name == id {
				return fmt.Errorf("remove the project VM before deleting retained data (VM status: %s)", vm.Status)
			}
		}
		data, err := paths.ProjectData(id, false)
		if err != nil {
			return err
		}
		answer, err := prompt.ReadLine(ctx, fmt.Sprintf("Delete all retained data for %s? Type the project identifier: ", id))
		if err != nil {
			return err
		}
		if answer != id {
			return fmt.Errorf("data removal cancelled")
		}
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		if err = os.RemoveAll(filepath.Dir(data)); err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "Removed retained data for %s\n", id)
		return err
	}}
}
