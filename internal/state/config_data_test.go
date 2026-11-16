package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vbx/internal/config"
)

func TestConfigurationDataBinding(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_DATA_HOME", base)
	id := "vbx-project-123456abcdef"
	c := config.Defaults()
	c.Folder = "/workspaces/project"
	c.Persistence = []config.PersistencePath{{Guest: "~/.tool/sessions", Data: "sessions"}}
	paths := Paths{}
	r, err := paths.AllocateData(id, c, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(base, "vbx/projects", id, "config")
	if len(r.Bindings) != 1 || r.Bindings[0].Module != "" || r.Bindings[0].Host != want || r.Bindings[0].Guest != "/mnt/vbx-config-data" || r.Bindings[0].Source != "mount1" {
		t.Fatal(r)
	}
	if err = ValidateData(id, r); err != nil {
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
	bindings, mappings := saved.GuestRecords()
	if !strings.HasPrefix(bindings, "@config\t/mnt/vbx-config-data\t") || mappings != "@config\t~/.tool/sessions\tsessions\n" {
		t.Fatal(bindings, mappings)
	}
	if err = os.WriteFile(filepath.Join(want, "history"), []byte("valuable"), 0600); err != nil {
		t.Fatal(err)
	}
	again, err := paths.AllocateData(id, c, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if again.Bindings[0] != saved.Bindings[0] {
		t.Fatal(again)
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err = ValidateData(id, saved); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(want, 0755); err != nil {
		t.Fatal(err)
	}
	if err = ValidateData(id, saved); err == nil {
		t.Fatal("public configuration data directory accepted")
	}
}
