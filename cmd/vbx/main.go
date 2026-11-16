package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"vbx"
	"vbx/internal/backend/lima"
	"vbx/internal/cli"
	"vbx/internal/command"
	"vbx/internal/commands"
	"vbx/internal/host"
	"vbx/internal/interaction"
	"vbx/internal/process"
	"vbx/internal/state"
	"vbx/internal/workspace"
)

func run() int {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)
	go func() {
		select {
		case sig := <-signals:
			cancel(process.SignalError{Signal: sig.(syscall.Signal)})
		case <-ctx.Done():
		}
	}()
	runner := process.OS{}
	paths := state.Paths{}
	terminal := interaction.Terminal{In: os.Stdin, Out: os.Stderr, IsTTY: func() bool { return process.IsTerminal(os.Stdin.Fd()) }}
	backend := &lima.Client{Runner: runner, In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
	hostChecks := host.Checker{}
	workspaces := &workspace.Service{Backend: backend, Host: hostChecks, Interaction: terminal, Assets: vbx.Assets, Paths: paths}
	cmds := commands.New(commands.Dependencies{Ready: workspaces, Scopes: workspaces, Inventory: backend, Lima: backend, Replacer: runner, Runner: runner, Host: hostChecks, Interaction: terminal, Paths: paths, Out: os.Stdout, Err: os.Stderr})
	e := cli.Run(ctx, os.Args[1:], cmds, os.Stdout)
	if e == nil {
		return 0
	}
	fmt.Fprintln(os.Stderr, "ERROR:", e)
	if ctx.Err() != nil {
		var cause process.SignalError
		if errors.As(context.Cause(ctx), &cause) {
			return cause.ExitCode()
		}
		return 1
	}
	var u *command.UsageError
	if errors.As(e, &u) {
		return 2
	}
	return 1
}
func main() { os.Exit(run()) }
