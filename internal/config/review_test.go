package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExplicitEmptyFolder(t *testing.T) {
	for _, value := range []string{"''", `""`} {
		t.Run(value, func(t *testing.T) {
			local, p := t.TempDir(), t.TempDir()
			os.WriteFile(filepath.Join(local, "config.yaml"), []byte("version: 1\nlaunch:\n  folder: "+value+"\n"), 0600)
			if _, err := Resolve(p, "/workspaces/app", local); err == nil {
				t.Fatal("empty folder accepted")
			}
			os.Mkdir(filepath.Join(p, ".vbx"), 0700)
			os.WriteFile(filepath.Join(p, ".vbx/config.yaml"), []byte("version: 1\nlaunch:\n  folder: /workspaces/override\n"), 0600)
			c, err := Resolve(p, "/workspaces/app", local)
			if err != nil || c.Folder != "/workspaces/override" {
				t.Fatal(c, err)
			}
		})
	}
}
