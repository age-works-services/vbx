package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"vbx"
	"vbx/internal/backend/lima"
	"vbx/internal/config"
	"vbx/internal/module"
	"vbx/internal/project"
	"vbx/internal/state"
)

func TestGuestPersistenceOperations(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	store := filepath.Join(root, "store")
	records := filepath.Join(root, "records")
	bin := filepath.Join(root, "bin")
	for _, dir := range []string{home, store, records, bin} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	script, err := vbx.Assets.ReadFile("scripts/persistence.sh")
	if err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(root, "persistence.sh")
	os.WriteFile(scriptPath, script, 0600)
	os.WriteFile(filepath.Join(records, "bindings.tsv"), []byte("sessions\t"+store+"\tmount1\tlima-34ff09956ef2385f\n"), 0600)
	os.WriteFile(filepath.Join(records, "mappings.tsv"), []byte("sessions\t~/.tool/sessions\tsessions\n"), 0600)
	os.WriteFile(filepath.Join(bin, "findmnt"), []byte("#!/bin/sh\n[ \"${TEST_MISSING_MOUNT:-0}\" = 0 ] || exit 1\nprintf '%s %s %s\\n' \"${TEST_MOUNT_TYPE:-9p}\" \"${TEST_MOUNT_SOURCE:-mount1}\" \"${TEST_MOUNT_TARGET:-$3}\"\n"), 0700)
	t.Setenv("HOME", home)
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	run := func(mode string, ok bool) {
		t.Helper()
		output, err := exec.Command("bash", scriptPath, mode, records, "vbx-fixture-123456abcdef").CombinedOutput()
		if (err == nil) != ok {
			t.Fatalf("%s: %v %s", mode, err, output)
		}
		if !ok && !strings.Contains(string(output), "limactl shell --workdir=/") {
			t.Fatal(string(output))
		}
	}
	run("apply", true)
	guest := filepath.Join(home, ".tool/sessions")
	target := filepath.Join(store, "sessions")
	os.WriteFile(filepath.Join(guest, "session"), []byte("history"), 0600)
	run("apply", true)
	run("verify", true)
	if value, _ := os.ReadFile(filepath.Join(target, "session")); string(value) != "history" {
		t.Fatal(string(value))
	}
	// A fresh guest home reconnects the same retained session directory.
	os.Remove(guest)
	os.Remove(filepath.Dir(guest))
	run("apply", true)
	run("verify", true)
	os.Remove(guest)
	os.Mkdir(guest, 0700)
	os.WriteFile(filepath.Join(guest, "local-session"), []byte("valuable"), 0600)
	run("verify", false)
	run("apply", false)
	if _, err := os.Stat(filepath.Join(guest, "local-session")); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(guest, "local-session"))
	os.Remove(guest)
	run("apply", true)
	os.Remove(filepath.Join(target, "session"))
	os.Remove(target)
	run("verify", false)
	os.Mkdir(target, 0700)
	t.Setenv("TEST_MISSING_MOUNT", "1")
	run("mounts", false)
	t.Setenv("TEST_MISSING_MOUNT", "0")
	t.Setenv("TEST_MOUNT_SOURCE", "mount0")
	run("mounts", false)
	t.Setenv("TEST_MOUNT_SOURCE", "lima-34ff09956ef2385f")
	run("mounts", true)
	run("apply", true)
	run("verify", true)
	t.Setenv("TEST_MOUNT_SOURCE", "lima-c0e08dd593f9922b")
	run("mounts", false)
	t.Setenv("TEST_MOUNT_SOURCE", "mount1")
	t.Setenv("TEST_MOUNT_TYPE", "ext4")
	run("mounts", false)
	t.Setenv("TEST_MOUNT_TYPE", "9p")
	t.Setenv("TEST_MOUNT_TARGET", "/")
	run("mounts", false)
	t.Setenv("TEST_MOUNT_TARGET", store)
	os.Rename(filepath.Dir(guest), filepath.Dir(guest)+"-real")
	os.Symlink(filepath.Dir(guest)+"-real", filepath.Dir(guest))
	run("verify", false)
	os.Remove(filepath.Dir(guest))
	os.Rename(filepath.Dir(guest)+"-real", filepath.Dir(guest))
	run("verify", true)
}

type persistenceBackend struct {
	backendStub
	instances []lima.Instance
	starts    int
}

func (b *persistenceBackend) List(context.Context) ([]lima.Instance, error) { return b.instances, nil }
func (b *persistenceBackend) Start(context.Context, string, string) error   { b.starts++; return nil }
func TestReuseRejectsStorageBeforeStart(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, err := project.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := config.Defaults()
	c.Folder = p.Primary
	c.Modules = []string{"sessions"}
	paths := state.Paths{}
	r, err := paths.AllocateData(p.Name, c, []module.Definition{{Name: "sessions"}}, p.Path)
	if err != nil {
		t.Fatal(err)
	}
	instance, _ := paths.Instance(p.Name)
	os.MkdirAll(instance.Dir, 0700)
	data, err := state.EncodeRecord(r)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(instance.Dir, "config"), data, 0600)
	os.WriteFile(filepath.Join(instance.Dir, "status"), []byte("ready\n"), 0600)
	b := &persistenceBackend{backendStub: backendStub{exists: true}, instances: []lima.Instance{{Name: p.Name, MountType: "9p", Mounts: []lima.Mount{{Location: p.Path, MountPoint: p.Primary, Writable: true}, {Location: r.Bindings[0].Host, MountPoint: r.Bindings[0].Guest, Writable: false}}}}}
	s := Service{Backend: b, Host: hostStub{}, Paths: paths, Assets: vbx.Assets}
	if _, err = s.EnsureReady(context.Background(), p.Path); err == nil || b.starts != 0 {
		t.Fatal("changed mount accepted", err, b.starts)
	}
	b.instances[0].Mounts[1].Writable = true
	os.Remove(r.Bindings[0].Host)
	if _, err = s.EnsureReady(context.Background(), p.Path); err == nil || b.starts != 0 {
		t.Fatal("missing data accepted", err, b.starts)
	}
	if _, err = os.Stat(r.Bindings[0].Host); !os.IsNotExist(err) {
		t.Fatal("source recreated", err)
	}
}
