package commands

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vbx"
	"vbx/internal/backend/lima"
	"vbx/internal/command"
	"vbx/internal/interaction"
	"vbx/internal/process"
	"vbx/internal/project"
	"vbx/internal/state"
	"vbx/internal/workspace"
)

type fake struct {
	calls               []string
	exists, status, ssh string
	fail                string
	replacement         []string
	archives            [][]byte
	hook                func(context.Context, []string) error
}

func (f *fake) Run(ctx context.Context, in io.Reader, out, errOut io.Writer, name string, args ...string) error {
	s := strings.Join(args, " ")
	f.calls = append(f.calls, s)
	if f.hook != nil {
		if err := f.hook(ctx, args); err != nil {
			return err
		}
	}
	if f.fail != "" && strings.HasPrefix(s, f.fail) {
		return fmt.Errorf("fixture failure")
	}
	switch args[0] {
	case "list":
		switch args[2] {
		case "{{.Name}}":
			fmt.Fprintln(out, f.exists)
		case "{{.Status}}":
			fmt.Fprintln(out, f.status)
		default:
			fmt.Fprintln(out, f.ssh)
		}
	case "start":
		if strings.HasPrefix(args[2], "--name=") {
			f.exists = strings.TrimPrefix(args[2], "--name=")
		}
	case "shell":
		b, _ := io.ReadAll(in)
		f.archives = append(f.archives, b)
	case "delete":
		f.exists = ""
	}
	return nil
}
func (f *fake) Replace(name string, args []string) error {
	f.replacement = append([]string{name}, args...)
	return nil
}

type testEnvironment struct {
	Assets    fs.FS
	Runner    process.Runner
	In        io.Reader
	Out, Err  io.Writer
	IsTTY     func() bool
	Preflight func(context.Context, bool) error
}
type hostFunc func(context.Context) error

func (f hostFunc) Check(ctx context.Context) error { return f(ctx) }
func (a *testEnvironment) dependencies() Dependencies {
	backend := &lima.Client{Runner: a.Runner, In: a.In, Out: a.Out, Err: a.Err}
	prompt := interaction.Terminal{In: a.In, Out: a.Err, IsTTY: a.IsTTY}
	host := hostFunc(func(ctx context.Context) error { return a.Preflight(ctx, true) })
	service := &workspace.Service{Backend: backend, Host: host, Interaction: prompt, Assets: a.Assets, Paths: state.Paths{}}
	return Dependencies{Ready: service, Scopes: service, Inventory: backend, Lima: backend, Replacer: a.Runner, Runner: a.Runner, Host: host, Interaction: prompt, Paths: state.Paths{}, Out: a.Out, Err: a.Err}
}
func runStart(ctx context.Context, a *testEnvironment, path, target string) error {
	flags := map[string]bool{}
	if target != "" {
		flags[target] = true
	}
	return newStartCommand(a.dependencies()).Run(ctx, command.NewInvocation(command.GeneralOptions{Project: path}, flags, nil))
}
func runExec(ctx context.Context, a *testEnvironment, path string, args []string) error {
	return newExecCommand(a.dependencies()).Run(ctx, command.NewInvocation(command.GeneralOptions{Project: path}, nil, args))
}
func runStop(ctx context.Context, a *testEnvironment, path string) error {
	return newStopCommand(a.dependencies()).Run(ctx, command.NewInvocation(command.GeneralOptions{Project: path}, nil, nil))
}
func runRemove(ctx context.Context, a *testEnvironment, path string) error {
	return newRemoveCommand(a.dependencies()).Run(ctx, command.NewInvocation(command.GeneralOptions{Project: path}, nil, nil))
}
func localDir() (string, error) { return (state.Paths{}).ConfigDir() }
func fixture(t *testing.T) (*testEnvironment, *fake, project.Project, string) {
	t.Helper()
	d := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(d, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(d, "state"))
	p, e := project.Resolve(d)
	if e != nil {
		t.Fatal(e)
	}
	ssh := filepath.Join(d, "ssh.config")
	os.WriteFile(ssh, []byte("Host lima-fixture\n"), 0600)
	f := &fake{ssh: ssh, status: "Stopped"}
	a := &testEnvironment{Assets: vbx.Assets}
	a.Runner = f
	a.Out = io.Discard
	a.Err = io.Discard
	a.In = strings.NewReader("")
	a.IsTTY = func() bool { return true }
	a.Preflight = func(context.Context, bool) error { return nil }
	return a, f, p, filepath.Join(d, "state/vbx/instances", p.Name)
}
