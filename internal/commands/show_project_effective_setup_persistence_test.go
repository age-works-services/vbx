package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vbx/internal/cli"
)

func TestShowSetupMergedPersistence(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	local, project := filepath.Join(root, "config/vbx"), filepath.Join(root, "project")
	for _, dir := range []string{local, filepath.Join(project, ".vbx")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for path, data := range map[string]string{
		filepath.Join(local, "config.yaml"):             "version: 1\nmodules:\n  - vscode-project-persistent-state\npersistence:\n  paths:\n    - guest: ~/.vscode-server/extensions\n      data: user-extensions\n",
		filepath.Join(project, ".vbx/persistence.yaml"): "persistence:\n  paths:\n    - guest: ~/.vscode-server/data/Machine\n      data: project-machine\n",
	} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var out strings.Builder
	if err := cli.Run(context.Background(), []string{"show-project-effective-setup", "--project", project}, New(Dependencies{Out: &out}), &out); err != nil {
		t.Fatal(err)
	}
	for _, guest := range []string{"~/.vscode-server/extensions", "~/.vscode-server/data/Machine"} {
		if strings.Count(out.String(), guest) != 1 {
			t.Fatal(out.String())
		}
	}
	for _, data := range []string{"user-extensions", "project-machine"} {
		if !strings.Contains(out.String(), data) {
			t.Fatal(out.String())
		}
	}
	if _, err := os.Stat(filepath.Join(root, "data")); !os.IsNotExist(err) {
		t.Fatal("inspection allocated persistence storage", err)
	}
}
