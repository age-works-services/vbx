package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vbx/internal/config"
	"vbx/internal/state"
)

func TestLifecycle(t *testing.T) {
	a, f, p, dir := fixture(t)
	ctx := context.Background()
	if e := runStart(ctx, a, p.Path, "workspace-only"); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(dir, "status"))
	if e != nil || string(b) != "ready\n" || len(f.archives) != 1 {
		t.Fatal(string(b), e)
	}
	if _, e = os.Stat(filepath.Join(dir, "provision.tar")); !os.IsNotExist(e) {
		t.Fatal("archive retained")
	} // Changed invalid desired configuration cannot affect reuse.
	os.Mkdir(filepath.Join(p.Path, ".vbx"), 0700)
	os.WriteFile(filepath.Join(p.Path, ".vbx/config.yaml"), []byte("broken: yaml\n"), 0600)
	for _, target := range []string{"", "terminal", "workspace-only"} {
		if e = runStart(ctx, a, p.Path, target); e != nil {
			t.Fatal(e)
		}
	}
	if len(f.archives) != 1 {
		t.Fatal("reprovisioned")
	}
	if e = runExec(ctx, a, p.Path, []string{"printf", "a b; $(touch /unexpected)"}); e != nil {
		t.Fatal(e)
	}
	if f.replacement[len(f.replacement)-1] != "a b; $(touch /unexpected)" {
		t.Fatal("argv changed")
	}
	if e = runStop(ctx, a, p.Path); e != nil {
		t.Fatal(e)
	}
	a.In = strings.NewReader(p.Name + "\n")
	f.fail = "delete"
	if e = runRemove(ctx, a, p.Path); e == nil {
		t.Fatal("delete succeeded")
	}
	if _, e = os.Stat(filepath.Join(dir, "config")); e != nil {
		t.Fatal("state removed on failed delete")
	}
	f.fail = ""
	a.In = strings.NewReader(p.Name + "\n")
	if e = runRemove(ctx, a, p.Path); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(dir); !os.IsNotExist(e) {
		t.Fatal("state remains")
	}
	if _, e = os.Stat(p.Path); e != nil {
		t.Fatal("project removed")
	}
}
func TestFrozenRetryAndLock(t *testing.T) {
	a, f, p, dir := fixture(t)
	ctx := context.Background()
	f.fail = "shell"
	if e := runStart(ctx, a, p.Path, "workspace-only"); e == nil {
		t.Fatal("setup succeeded")
	}
	before, e := os.ReadFile(filepath.Join(dir, "provision.tar"))
	if e != nil {
		t.Fatal(e)
	}
	base, _ := state.Base("XDG_STATE_HOME", ".local/state")
	unlock, e := state.Lock(base, p.Name)
	if e != nil {
		t.Fatal("lock retained", e)
	}
	if e = runStart(ctx, a, p.Path, "workspace-only"); e == nil {
		t.Fatal("concurrent operation accepted")
	}
	unlock()
	os.Mkdir(filepath.Join(p.Path, ".vbx"), 0700)
	os.WriteFile(filepath.Join(p.Path, ".vbx/config.yaml"), []byte("broken: yaml\n"), 0600)
	f.fail = ""
	if e = runStart(ctx, a, p.Path, "workspace-only"); e != nil {
		t.Fatal(e)
	}
	if string(before) != string(f.archives[0]) {
		t.Fatal("retry changed inputs")
	}
}
func TestMissingVMStateUserControlled(t *testing.T) {
	a, f, p, dir := fixture(t)
	f.fail = "start"
	if e := runStart(context.Background(), a, p.Path, "workspace-only"); e == nil {
		t.Fatal("start succeeded")
	}
	f.fail = ""
	f.calls = nil
	if e := runStart(context.Background(), a, p.Path, "workspace-only"); e == nil {
		t.Fatal("automatic repair")
	}
	for _, s := range f.calls {
		if strings.HasPrefix(s, "start") {
			t.Fatal("replaced vm")
		}
	}
	a.In = strings.NewReader(p.Name + "\n")
	if e := runRemove(context.Background(), a, p.Path); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(dir); !os.IsNotExist(e) {
		t.Fatal("leftover state not removable")
	}
}
func TestReadySnapshotReuse(t *testing.T) {
	a, f, p, dir := fixture(t)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	c := config.Defaults()
	c.Default = "workspace-only"
	c.Folder = p.Primary
	b, err := state.Encode(c)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "config"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "status"), []byte("ready\n"), 0600); err != nil {
		t.Fatal(err)
	}
	f.exists = p.Name
	if err = runStart(context.Background(), a, p.Path, ""); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(dir, "config"))
	if err != nil || string(after) != string(b) {
		t.Fatal("saved snapshot changed", err)
	}
}

func TestNoninteractiveApproval(t *testing.T) {
	a, f, p, _ := fixture(t)
	a.IsTTY = func() bool { return false }
	os.Mkdir(filepath.Join(p.Path, ".vbx"), 0700)
	if e := runStart(context.Background(), a, p.Path, ""); e == nil {
		t.Fatal("approval bypassed")
	}
	for _, s := range f.calls {
		if strings.HasPrefix(s, "start") {
			t.Fatal("started without approval")
		}
	}
}
