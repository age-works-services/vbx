package workspace

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"vbx"
	"vbx/internal/backend/lima"
	"vbx/internal/project"
	"vbx/internal/state"
)

type configDataBackend struct {
	backendStub
	recordPath  string
	p           project.Project
	validations int
	inputs      [][]byte
}

func (b *configDataBackend) Validate(context.Context, string) error      { b.validations++; return nil }
func (b *configDataBackend) Start(context.Context, string, string) error { b.exists = true; return nil }
func (b *configDataBackend) Shell(_ context.Context, _, _ string, in io.Reader, _ ...string) error {
	data, err := io.ReadAll(in)
	b.inputs = append(b.inputs, data)
	return err
}
func (b *configDataBackend) List(context.Context) ([]lima.Instance, error) {
	data, err := os.ReadFile(b.recordPath)
	if err != nil {
		return nil, err
	}
	r, err := state.DecodeRecord(data)
	if err != nil {
		return nil, err
	}
	mounts := []lima.Mount{{Location: b.p.Path, MountPoint: b.p.Primary, Writable: true}}
	for _, m := range r.Config.Mounts {
		mounts = append(mounts, lima.Mount{Location: m.Source, MountPoint: m.Target, Writable: !m.Readonly})
	}
	for _, m := range r.Bindings {
		mounts = append(mounts, lima.Mount{Location: m.Host, MountPoint: m.Guest, Writable: true})
	}
	return []lima.Instance{{Name: b.p.Name, Status: "Running", MountType: "9p", Mounts: mounts}}, nil
}

type approveConfigData struct{}

func (approveConfigData) ReadLine(context.Context, string) (string, error) { return "yes", nil }
func TestCreateCapturesAndReusesConfigurationPersistence(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	p, err := project.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(p.Path, ".vbx/persistence.yaml")
	if err = os.MkdirAll(filepath.Dir(source), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(source, []byte("persistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: sessions\n"), 0600); err != nil {
		t.Fatal(err)
	}
	paths := state.Paths{}
	instance, err := paths.Instance(p.Name)
	if err != nil {
		t.Fatal(err)
	}
	ssh := filepath.Join(root, "ssh")
	if err = os.WriteFile(ssh, []byte("Host fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	b := &configDataBackend{backendStub: backendStub{ssh: ssh}, recordPath: filepath.Join(instance.Dir, "config"), p: p}
	service := Service{Backend: b, Host: hostStub{}, Paths: paths, Assets: vbx.Assets, Interaction: approveConfigData{}}
	ready, err := service.EnsureReady(context.Background(), p.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(ready.Config.Persistence) != 1 || ready.Config.Persistence[0].Data != "sessions" || b.validations != 1 || len(b.inputs) != 3 {
		t.Fatal(ready.Config, b.validations, len(b.inputs))
	}
	data, err := os.ReadFile(b.recordPath)
	if err != nil {
		t.Fatal(err)
	}
	record, err := state.DecodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Bindings) != 1 || record.Bindings[0].Guest != "/mnt/vbx-config-data" || record.Mappings[0].Module != "" {
		t.Fatal(record)
	}
	if err = os.WriteFile(source, []byte("invalid current desired configuration"), 0600); err != nil {
		t.Fatal(err)
	}
	ready, err = service.EnsureReady(context.Background(), p.Path)
	if err != nil {
		t.Fatal(err)
	}
	if ready.Config.Persistence[0].Data != "sessions" || len(b.inputs) != 5 || b.validations != 1 {
		t.Fatal("reuse ignored frozen mappings", ready.Config)
	}
}
