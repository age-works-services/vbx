package state

import (
	"os"
	"path/filepath"
	"testing"
	"vbx/internal/config"
	"vbx/internal/module"
)

func dataFixture(t *testing.T) (Paths, string, Record) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	paths := Paths{}
	id := "vbx-project-123456abcdef"
	c := config.Defaults()
	c.Folder = "/workspaces/project"
	c.Modules = []string{"sessions"}
	c.Persistence = []config.PersistencePath{{Module: "sessions", Guest: "~/.tool/sessions", Data: "sessions"}}
	r, err := paths.AllocateData(id, c, []module.Definition{{Name: "sessions", Paths: []module.Path{{Guest: "~/.tool/sessions", Data: "sessions"}}}}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(os.Getenv("XDG_DATA_HOME"), "vbx", "projects", id, "modules", "sessions"); r.Bindings[0].Host != want {
		t.Fatalf("data path %q, want %q", r.Bindings[0].Host, want)
	}
	return paths, id, r
}
func TestPersistentDataIdentityAndSnapshot(t *testing.T) {
	paths, id, r := dataFixture(t)
	if err := ValidateData(id, r); err != nil {
		t.Fatal(err)
	}
	data, err := EncodeRecord(r)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := DecodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(r.Bindings[0].Host, "session")
	if err = os.WriteFile(marker, []byte("history"), 0600); err != nil {
		t.Fatal(err)
	}
	again, err := paths.AllocateData(id, r.Config, []module.Definition{{Name: "sessions", Paths: []module.Path{{Guest: "~/.tool/sessions", Data: "sessions"}}}}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if again.Bindings[0] != r.Bindings[0] {
		t.Fatal(again)
	}
	if _, err = os.Stat(marker); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err = ValidateData(id, saved); err != nil {
		t.Fatal("captured binding changed with XDG", err)
	}
	if err = os.Chmod(r.Bindings[0].Host, 0755); err != nil {
		t.Fatal(err)
	}
	if err = ValidateData(id, r); err == nil {
		t.Fatal("public directory accepted")
	}
}
func TestMissingAndSymlinkData(t *testing.T) {
	_, id, r := dataFixture(t)
	host := r.Bindings[0].Host
	if err := os.Remove(host); err != nil {
		t.Fatal(err)
	}
	if err := ValidateData(id, r); err == nil {
		t.Fatal("missing data accepted")
	}
	if _, err := os.Lstat(host); !os.IsNotExist(err) {
		t.Fatal("missing source recreated", err)
	}
	if err := os.Symlink(t.TempDir(), host); err != nil {
		t.Fatal(err)
	}
	if err := ValidateData(id, r); err == nil {
		t.Fatal("symlink data accepted")
	}
}
func TestSnapshotMappingValidation(t *testing.T) {
	_, _, r := dataFixture(t)
	r.Mappings[0].Data = "../outside"
	if _, err := EncodeRecord(r); err == nil {
		t.Fatal("escape accepted")
	}
}
