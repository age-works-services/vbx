package module

import (
	"testing"
	"testing/fstest"
)

func TestPersistenceDefinitions(t *testing.T) {
	assets := fstest.MapFS{"modules/install/setup.sh": {Data: []byte("true")}, "modules/sessions/persistence.yaml": {Data: []byte("persistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: sessions\n")}}
	defs, err := Load(assets, []string{"install", "sessions"})
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 2 || !defs[0].Setup || defs[1].Setup || defs[1].Paths[0].Data != "sessions" {
		t.Fatal(defs)
	}
}
func TestInvalidDeclarations(t *testing.T) {
	for _, yaml := range []string{
		"persistence:\n  paths: []", "persistence:\n  paths:\n    - guest: ~/\n      data: a", "persistence:\n  paths:\n    - guest: ~/../escape\n      data: a", "persistence:\n  paths:\n    - guest: ~/.config\n      data: a", "persistence:\n  paths:\n    - guest: ~/ok\n      data: ../escape", "persistence:\n  paths:\n    - guest: ~/ok\n      data: a\n      unknown: x", "persistence:\n  paths:\n    - guest: ~/ok\n      guest: ~/other\n", "persistence:\n  paths:\n    - guest: ~/ok\n      data: $HOME"} {
		if _, err := Parse([]byte(yaml)); err == nil {
			t.Fatal("accepted", yaml)
		}
	}

}
func TestOverlayCollision(t *testing.T) {
	defs := []Definition{{Name: "sessions", Paths: []Path{{Guest: "~/.tool/sessions", Data: "sessions"}}}}
	for _, entry := range []struct {
		path string
		dir  bool
	}{{".tool/sessions", true}, {".tool/sessions/file", false}, {".tool", false}, {".config/vbx/environment.sh", false}, {".local/state/vbx/file", false}} {
		if err := CheckOverlay(entry.path, entry.dir, defs, map[string]bool{}); err == nil {
			t.Fatal(entry)
		}
	}
	kinds := map[string]bool{}
	if err := CheckOverlay(".tool", true, defs, kinds); err != nil {
		t.Fatal(err)
	}
	if err := CheckOverlay(".tool/settings.json", false, defs, kinds); err != nil {
		t.Fatal(err)
	}
	if err := CheckOverlay("link", false, defs, kinds); err != nil {
		t.Fatal(err)
	}
	if err := CheckOverlay("link/child", false, defs, kinds); err == nil {
		t.Fatal("overlay traversed symlink")
	}
	if err := CheckOverlay("link", true, defs, kinds); err == nil {
		t.Fatal("overlay replaced symlink ancestor")
	}
}
