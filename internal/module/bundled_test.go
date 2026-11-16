package module

import (
	"slices"
	"testing"
	"vbx"
)

func TestBundledDefinitions(t *testing.T) {
	defs, err := Load(vbx.Assets, []string{"aws-cli", "dev-tools", "node", "pi", "pi-web-fetch", "pi-project-persistent-sessions", "vscode-project-persistent-state"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Path{{Guest: "~/.vscode-server/extensions", Data: "extensions"}, {Guest: "~/.vscode-server/data/Machine", Data: "machine-settings"}}
	for i := range want {
		want[i].Module = "vscode-project-persistent-state"
	}
	vscode := defs[len(defs)-1]
	if vscode.Setup || !slices.Equal(vscode.Paths, want) {
		t.Fatalf("VS Code persistence mappings: %#v", vscode)
	}
	for _, d := range defs {
		if !d.Setup && len(d.Paths) == 0 {
			t.Fatal(d)
		}
	}
}
