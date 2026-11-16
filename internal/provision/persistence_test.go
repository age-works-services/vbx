package provision

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"vbx"
	"vbx/internal/config"
	"vbx/internal/state"
)

func TestCapturedPersistencePlan(t *testing.T) {
	manifest := "persistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: sessions\n"
	assets := fstest.MapFS{"modules/sessions/persistence.yaml": {Data: []byte(manifest)}}
	for _, name := range []string{"provision-guest.sh", "apt-install.sh", "guest-shell-system.sh", "npm-policy.sh", "persistence.sh"} {
		data, err := vbx.Assets.ReadFile("scripts/" + name)
		if err != nil {
			t.Fatal(err)
		}
		assets["scripts/"+name] = &fstest.MapFile{Data: data}
	}
	c := config.Defaults()
	c.Modules = []string{"sessions"}
	c.Persistence = []config.PersistencePath{{Module: "sessions", Guest: "~/.tool/sessions", Data: "sessions"}}
	r := bundleRecord(c)
	r.Mappings = []state.Mapping{{Module: "sessions", Guest: "~/.tool/sessions", Data: "sessions"}}
	var archive bytes.Buffer
	local, project := t.TempDir(), t.TempDir()
	if err := Build(&archive, assets, r, local, project); err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	reader := tar.NewReader(&archive)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		contents[header.Name] = string(data)
	}
	bindings, mappings := r.GuestRecords()
	if contents["bindings.tsv"] != bindings || contents["mappings.tsv"] != mappings || contents["modules/sessions/persistence.yaml"] != manifest || contents["persistence.sh"] == "" {
		t.Fatal(contents)
	}
	guest := filepath.Join(local, "home/.tool/sessions")
	if err := os.MkdirAll(guest, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(guest, "history"), []byte("valuable"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Build(io.Discard, assets, r, local, project); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatal("overlay collision accepted", err)
	}
	if data, err := os.ReadFile(filepath.Join(guest, "history")); err != nil || string(data) != "valuable" {
		t.Fatal("overlay input modified", err)
	}
	r.Mappings[0].Data = "different"
	if err := Build(io.Discard, assets, r, t.TempDir(), project); err == nil {
		t.Fatal("bundle/snapshot disagreement accepted")
	}
}
