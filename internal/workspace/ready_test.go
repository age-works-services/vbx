package workspace

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"vbx/internal/backend/lima"
	"vbx/internal/config"
	"vbx/internal/project"
	"vbx/internal/state"
)

type backendStub struct {
	calls        []string
	exists       bool
	ssh, archive string
	fail         error
	locked       func()
}

func (b *backendStub) Exists(context.Context, string) (bool, error) {
	b.calls = append(b.calls, "exists")
	if b.locked != nil {
		b.locked()
	}
	return b.exists, nil
}
func (b *backendStub) Start(_ context.Context, _ string, template string) error {
	b.calls = append(b.calls, "start")
	if template != "" {
		return errors.New("unexpected creation")
	}
	return nil
}
func (b *backendStub) Validate(context.Context, string) error {
	return errors.New("unexpected validation")
}
func (b *backendStub) Shell(_ context.Context, _, _ string, in io.Reader, _ ...string) error {
	b.calls = append(b.calls, "setup")
	data, err := io.ReadAll(in)
	b.archive = string(data)
	if err != nil {
		return err
	}
	return b.fail
}
func (b *backendStub) SSHConfigFile(context.Context, string) (string, error) {
	b.calls = append(b.calls, "ssh")
	return b.ssh, nil
}

func (b *backendStub) List(context.Context) ([]lima.Instance, error) { return nil, nil }

type hostStub struct{}

func (hostStub) Check(context.Context) error { return nil }
func readyFixture(t *testing.T) (*Service, *backendStub, project.Project, state.InstancePaths) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	p, err := project.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	paths := state.Paths{}
	instance, err := paths.Instance(p.Name)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(instance.Dir, 0700); err != nil {
		t.Fatal(err)
	}
	c := config.Config{CPUs: "2", Memory: "4GiB", Disk: "20GiB", Default: "terminal", Folder: p.Primary}
	data, err := state.Encode(c)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"config": data, "status": []byte("setup-failed\n"), "provision.tar": []byte("frozen credentials")} {
		if err = os.WriteFile(filepath.Join(instance.Dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	ssh := filepath.Join(t.TempDir(), "ssh")
	if err = os.WriteFile(ssh, []byte("Host test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	b := &backendStub{exists: true, ssh: ssh}
	return &Service{Backend: b, Host: hostStub{}, Paths: paths}, b, p, instance
}
func TestReadinessFrozenRetryAndReleasedScope(t *testing.T) {
	s, b, p, instance := readyFixture(t)
	base, err := s.Paths.StateDir()
	if err != nil {
		t.Fatal(err)
	}
	b.locked = func() {
		if unlock, err := state.Lock(base, p.Name); err == nil {
			unlock()
			t.Error("readiness did not hold scope lock")
		}
	}
	b.fail = errors.New("setup failed")
	if _, err = s.EnsureReady(context.Background(), p.Path); err == nil {
		t.Fatal("setup succeeded")
	}
	data, err := os.ReadFile(filepath.Join(instance.Dir, "status"))
	if err != nil || string(data) != "setup-failed\n" {
		t.Fatal(string(data), err)
	}
	b.fail = nil
	b.calls = nil
	// Existing workspaces use the saved snapshot, independently of desired inputs.
	local, _ := s.Paths.ConfigDir()
	if err = os.MkdirAll(local, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(local, "config.yaml"), []byte("invalid desired configuration"), 0600); err != nil {
		t.Fatal(err)
	}
	ready, err := s.EnsureReady(context.Background(), p.Path)
	if err != nil {
		t.Fatal(err)
	}
	if ready.Config.Default != "terminal" || b.archive != "frozen credentials" || !reflect.DeepEqual(b.calls, []string{"exists", "start", "setup", "ssh"}) {
		t.Fatal(ready, b.calls, b.archive)
	}
	if _, err = os.Stat(filepath.Join(instance.Dir, "provision.tar")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	unlock, err := state.Lock(base, p.Name)
	if err != nil {
		t.Fatal("scope retained after readiness", err)
	}
	unlock()
	b.calls = nil
	if _, err = s.EnsureReady(context.Background(), p.Path); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b.calls, []string{"exists", "start", "ssh"}) {
		t.Fatal(b.calls)
	}
}
func TestScopeOrphanAndIdempotentRelease(t *testing.T) {
	s, b, p, _ := readyFixture(t)
	b.exists = false
	scope, err := s.Acquire(context.Background(), p.Path)
	if err != nil {
		t.Fatal(err)
	}
	if scope.VMExists || !scope.StateExists {
		t.Fatal(scope)
	}
	scope.Release()
	scope.Release()
	if _, err = s.EnsureReady(context.Background(), p.Path); err == nil {
		t.Fatal("orphan accepted")
	}
	if !reflect.DeepEqual(b.calls, []string{"exists", "exists"}) {
		t.Fatal(b.calls)
	}
}
