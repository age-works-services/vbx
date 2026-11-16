package process

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Runner interface {
	Run(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error
	Replace(string, []string) error
}
type OS struct{}

func (OS) Run(ctx context.Context, in io.Reader, out, errOut io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	// Lifecycle children use a separate process group. Forward each cancellation
	// once; final interactive handoff uses Replace instead.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		sig := syscall.SIGTERM
		var cause SignalError
		if errors.As(context.Cause(ctx), &cause) {
			sig = cause.Signal
		}
		e := syscall.Kill(-cmd.Process.Pid, sig)
		if errors.Is(e, syscall.ESRCH) || errors.Is(e, os.ErrProcessDone) {
			return os.ErrProcessDone
		}
		return e
	}
	cmd.WaitDelay = 5 * time.Second
	return cmd.Run()
}
func (OS) Replace(name string, args []string) error {
	path, e := exec.LookPath(name)
	if e != nil {
		return e
	}
	return syscall.Exec(path, append([]string{name}, args...), os.Environ())
}
