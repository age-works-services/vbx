package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"vbx/internal/backend/lima"
	"vbx/internal/command"
	"vbx/internal/config"
	"vbx/internal/project"
	"vbx/internal/workspace"
)

type readyStub struct {
	result workspace.Ready
	err    error
	calls  int
	cancel context.CancelFunc
}

func (s *readyStub) EnsureReady(context.Context, string) (workspace.Ready, error) {
	s.calls++
	if s.cancel != nil {
		s.cancel()
	}
	return s.result, s.err
}

type handoffStub struct {
	name string
	args []string
}

func (s *handoffStub) Replace(name string, args []string) error {
	s.name = name
	s.args = append([]string(nil), args...)
	return nil
}
func TestStartLaunchPolicy(t *testing.T) {
	for _, tc := range []struct{ saved, override, target string }{{"terminal", "", "terminal"}, {"vscode", "", "vscode"}, {"workspace-only", "", "workspace-only"}, {"terminal", "vscode", "vscode"}, {"vscode", "terminal", "terminal"}, {"terminal", "workspace-only", "workspace-only"}} {
		t.Run(tc.saved+"/"+tc.override, func(t *testing.T) {
			ready := &readyStub{result: workspace.Ready{Project: project.Project{Name: "vm", Primary: "/workspaces/p"}, Config: config.Config{Default: tc.saved, Folder: "/workspaces/p/folder"}}}
			handoff := &handoffStub{}
			var out strings.Builder
			c := newStartCommand(Dependencies{Ready: ready, Replacer: handoff, Out: &out})
			flags := map[string]bool{}
			if tc.override != "" {
				flags[tc.override] = true
			}
			if err := c.Run(context.Background(), command.NewInvocation(command.GeneralOptions{Project: "project"}, flags, nil)); err != nil {
				t.Fatal(err)
			}
			switch tc.target {
			case "terminal":
				if handoff.name != "limactl" || !reflect.DeepEqual(handoff.args, lima.ShellArgv("vm", "/workspaces/p", []string{"bash", "-l"})) {
					t.Fatal(handoff)
				}
			case "vscode":
				if handoff.name != "code" || !reflect.DeepEqual(handoff.args, []string{"--remote", "ssh-remote+lima-vm", "/workspaces/p/folder"}) {
					t.Fatal(handoff)
				}
			case "workspace-only":
				if handoff.name != "" || !strings.Contains(out.String(), "Ready: vm") {
					t.Fatal(handoff, out.String())
				}
			}
			if ready.result.Config.Default != tc.saved {
				t.Fatal("saved launch target changed")
			}
		})
	}
}
func TestHandoffChecksReadinessCancellation(t *testing.T) {
	for _, factory := range []Factory{newStartCommand, newExecCommand} {
		ctx, cancel := context.WithCancel(context.Background())
		ready := &readyStub{cancel: cancel}
		handoff := &handoffStub{}
		c := factory(Dependencies{Ready: ready, Replacer: handoff})
		err := c.Run(ctx, command.NewInvocation(command.GeneralOptions{}, nil, []string{"true"}))
		if !errors.Is(err, context.Canceled) || handoff.name != "" {
			t.Fatal(err, handoff)
		}
	}
}

type scopeStub struct{ s *workspace.Scope }

func (s scopeStub) Acquire(context.Context, string) (*workspace.Scope, error) { return s.s, nil }

type promptStub string

func (p promptStub) ReadLine(context.Context, string) (string, error) { return string(p), nil }

type lifecycleStub struct {
	calls []string
	fail  error
}

func (b *lifecycleStub) Status(context.Context, string) (string, error) {
	b.calls = append(b.calls, "status")
	return "Running", nil
}
func (b *lifecycleStub) Stop(context.Context, string) error {
	b.calls = append(b.calls, "stop")
	return nil
}
func (b *lifecycleStub) Delete(context.Context, string) error {
	b.calls = append(b.calls, "delete")
	return b.fail
}
func TestRemoveFailurePreservesLocalState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "instance")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ssh := filepath.Join(t.TempDir(), "ssh")
	if err := os.WriteFile(ssh, []byte("ssh"), 0600); err != nil {
		t.Fatal(err)
	}
	b := &lifecycleStub{fail: errors.New("delete failed")}
	c := newRemoveCommand(Dependencies{Scopes: scopeStub{&workspace.Scope{Project: project.Project{Name: "vm"}, InstanceDir: dir, SSHLink: ssh, VMExists: true, StateExists: true}}, Lima: b, Interaction: promptStub("vm")})
	if err := c.Run(context.Background(), command.Invocation{}); !errors.Is(err, b.fail) {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b.calls, []string{"status", "stop", "delete"}) {
		t.Fatal(b.calls)
	}
	for _, p := range []string{dir, ssh} {
		if _, err := os.Stat(p); err != nil {
			t.Fatal("cleanup preceded successful deletion", err)
		}
	}
}
