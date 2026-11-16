package commands

import (
	"context"
	"vbx/internal/backend/lima"
	"vbx/internal/command"
)

func init() { register(newExecCommand) }
func newExecCommand(d Dependencies) command.Command {
	ready, handoff := d.Ready, d.Replacer
	return command.Handler{Definition: command.Spec{Name: "exec", Summary: "Run a command in the guest", Usage: "exec [OPTIONS] [--] COMMAND [ARG...]", Operands: command.RequiredOperands, OperandHint: "a guest command"}, Execute: func(ctx context.Context, in command.Invocation) error {
		if len(in.Args) == 0 {
			return &command.UsageError{Message: "exec requires a guest command"}
		}
		w, err := ready.EnsureReady(ctx, in.General.Project)
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		return handoff.Replace("limactl", lima.ShellArgv(w.Project.Name, w.Project.Primary, in.Args))
	}}
}
