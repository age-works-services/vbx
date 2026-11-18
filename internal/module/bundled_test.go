package module

import (
	"io/fs"
	"slices"
	"testing"
	"vbx"
)

func TestBundledDefinitions(t *testing.T) {
	entries, err := fs.ReadDir(vbx.Assets, "modules")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			t.Run(entry.Name(), func(t *testing.T) {
				if _, err := Load(vbx.Assets, []string{entry.Name()}); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestVSCodePersistenceDefinition(t *testing.T) {
	defs, err := Load(vbx.Assets, []string{"vscode-project-persistent-state"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Path{{Guest: "~/.vscode-server/extensions", Data: "extensions"}, {Guest: "~/.vscode-server/data/Machine", Data: "machine-settings"}}
	for i := range want {
		want[i].Module = "vscode-project-persistent-state"
	}
	vscode := defs[0]
	if vscode.Setup || !slices.Equal(vscode.Paths, want) {
		t.Fatalf("VS Code persistence mappings: %#v", vscode)
	}
}
