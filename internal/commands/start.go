package commands

import (
	"context"
	"fmt"
	"vbx/internal/backend/lima"
	"vbx/internal/command"
)

func init() { register(newStartCommand) }
func newStartCommand(d Dependencies) command.Command {
	ready, handoff, out := d.Ready, d.Replacer, d.Out
	return command.Handler{Definition: command.Spec{Name: "start", Usage: "start [--vscode | --terminal | --workspace-only]", Summary: "Create/start the VM and launch the configured target", Flags: map[string]command.Flag{"vscode": {Summary: "Open VS Code"}, "terminal": {Summary: "Open a guest login shell"}, "workspace-only": {Summary: "Start without opening a tool"}}}, Execute: func(ctx context.Context, in command.Invocation) error {
		if in.Flags.Len() > 1 {
			return &command.UsageError{Message: "select only one launch option"}
		}
		target := ""
		for _, name := range []string{"vscode", "terminal", "workspace-only"} {
			if in.Flags.Bool(name) {
				target = name
			}
		}
		w, err := ready.EnsureReady(ctx, in.General.Project)
		if err != nil {
			return err
		}
		if target == "" {
			target = w.Config.Default
		}
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		switch target {
		case "workspace-only":
			_, err = fmt.Fprintf(out, "Ready: %s (%s)\n", w.Project.Name, w.Project.Primary)
			return err
		case "terminal":
			return handoff.Replace("limactl", lima.ShellArgv(w.Project.Name, w.Project.Primary, []string{"bash", "-l"}))
		case "vscode":
			return handoff.Replace("code", []string{"--remote", "ssh-remote+" + lima.SSHHost(w.Project.Name), w.Config.Folder})
		default:
			return fmt.Errorf("unknown launch target %q", target)
		}
	}}
}
