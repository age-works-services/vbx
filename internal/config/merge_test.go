package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func putSettings(t *testing.T, dir, name, data string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}
func moduleFixture() fstest.MapFS {
	return fstest.MapFS{
		"modules/node/config.yaml": {Data: []byte("version: 1\nresources:\n  cpus: 3\npersistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: node-sessions\n")},
		"modules/pi/config.yaml":   {Data: []byte("version: 1\nresources:\n  cpus: 4\npersistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: pi-sessions\n")},
	}
}
func TestModulesResolveFromAssets(t *testing.T) {
	local, project := t.TempDir(), t.TempDir()
	assets := fstest.MapFS{
		"modules/new-tool/setup.sh": {},
		"modules/new-tool/apt.txt":  {Data: []byte("make\n")},
		"modules/not-a-directory":   {},
	}
	putSettings(t, project, ".vbx/config.yaml", "version: 1\nmodules:\n  - new-tool\n")
	c, err := ResolveWithAssets(project, "/workspaces/project", local, assets)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(c.Modules, ",") != "new-tool" || strings.Join(c.APT, ",") != "make" {
		t.Fatalf("resolved modules %v and APT %v", c.Modules, c.APT)
	}
	for _, tc := range []struct{ name, message string }{
		{"node", "unknown bundled module"},
		{"not-a-directory", "unknown bundled module"},
		{"../new-tool", "invalid module identifier"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			putSettings(t, project, ".vbx/config.yaml", "version: 1\nmodules:\n  - "+tc.name+"\n")
			_, err := ResolveWithAssets(project, "/workspaces/project", local, assets)
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("got %v, want %q", err, tc.message)
			}
		})
	}
}

func TestFiveLayerPrecedence(t *testing.T) {
	for _, tc := range []struct{ name, global, project, cpu, data, owner string }{
		{"defaults", "version: 1\n", "version: 1\n", "2", "", ""},
		{"global modules", "version: 1\nmodules:\n  - node\n  - pi\n", "version: 1\n", "4", "pi-sessions", "pi"},
		{"user global", "version: 1\nmodules:\n  - node\n  - pi\nresources:\n  cpus: 5\npersistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: user-sessions\n", "version: 1\n", "5", "user-sessions", ""},
		{"project reselection", "version: 1\nmodules:\n  - node\nresources:\n  cpus: 5\n", "version: 1\nmodules:\n  - node\n", "3", "node-sessions", "node"},
		{"project modules", "version: 1\nmodules:\n  - node\nresources:\n  cpus: 5\npersistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: user-sessions\n", "version: 1\nmodules:\n  - node\n  - pi\n", "4", "pi-sessions", "pi"},
		{"project config", "version: 1\nmodules:\n  - node\nresources:\n  cpus: 5\n", "version: 1\nmodules:\n  - pi\nresources:\n  cpus: 8\npersistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: project-sessions\n", "8", "project-sessions", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			global, project := t.TempDir(), t.TempDir()
			putSettings(t, global, "config.yaml", tc.global)
			putSettings(t, project, ".vbx/config.yaml", tc.project)
			c, err := ResolveWithAssets(project, "/workspaces/project", global, moduleFixture())
			if err != nil {
				t.Fatal(err)
			}
			if c.CPUs != tc.cpu {
				t.Fatalf("CPU %s, want %s", c.CPUs, tc.cpu)
			}
			if tc.data == "" {
				if len(c.Persistence) != 0 {
					t.Fatal(c.Persistence)
				}
			} else if len(c.Persistence) != 1 || c.Persistence[0].Data != tc.data || c.Persistence[0].Module != tc.owner {
				t.Fatal(c.Persistence)
			}
		})
	}
}
func TestSidecarAndInlineShareSchema(t *testing.T) {
	global, project := t.TempDir(), t.TempDir()
	putSettings(t, global, "persistence.yaml", "persistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: global-sidecar\n")
	putSettings(t, global, "config.yaml", "version: 1\npersistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: global-inline\n")
	putSettings(t, project, ".vbx/persistence.yaml", "persistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: project-sidecar\n")
	c, err := Resolve(project, "/workspaces/project", global)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Persistence) != 1 || c.Persistence[0].Data != "project-sidecar" {
		t.Fatal(c.Persistence)
	}
	putSettings(t, project, ".vbx/config.yaml", "version: 1\npersistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: project-inline\n")
	c, err = Resolve(project, "/workspaces/project", global)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Persistence) != 1 || c.Persistence[0].Data != "project-inline" {
		t.Fatal(c.Persistence)
	}
	if !strings.Contains(string(Format(c)), "persistence:\n  paths:\n    - guest:") {
		t.Fatal(string(Format(c)))
	}
}
func TestEffectivePersistenceConflicts(t *testing.T) {
	for _, second := range []string{"~/.tool/sessions", "~/.tool/sessions/child"} {
		t.Run(second, func(t *testing.T) {
			global, project := t.TempDir(), t.TempDir()
			putSettings(t, global, "config.yaml", "version: 1\nmodules:\n  - node\n")
			putSettings(t, project, ".vbx/config.yaml", "version: 1\npersistence:\n  paths:\n    - guest: "+second+"\n      data: override\n")
			_, err := ResolveWithAssets(project, "/workspaces/project", global, moduleFixture())
			if (err == nil) != (second == "~/.tool/sessions") {
				t.Fatal(err)
			}
		})
	}
	// Lower-priority data collisions disappear when the final mapping uses another scope.
	assets := moduleFixture()
	assets["modules/node/config.yaml"] = &fstest.MapFile{Data: []byte("version: 1\npersistence:\n  paths:\n    - guest: ~/.a\n      data: shared\n    - guest: ~/.b\n      data: shared\n")}
	global, project := t.TempDir(), t.TempDir()
	putSettings(t, global, "config.yaml", "version: 1\nmodules:\n  - node\n")
	putSettings(t, project, ".vbx/config.yaml", "version: 1\npersistence:\n  paths:\n    - guest: ~/.a\n      data: configuration-a\n")
	if _, err := ResolveWithAssets(project, "/workspaces/project", global, assets); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(project, ".vbx/config.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveWithAssets(project, "/workspaces/project", global, assets); err == nil {
		t.Fatal("effective data overlap accepted")
	}
}
func TestMountsMergeByTarget(t *testing.T) {
	global, project, one, two := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	putSettings(t, global, "config.yaml", "version: 1\nmounts:\n  - source: '"+one+"'\n    target: /workspaces/extra\n    readonly: true\n")
	putSettings(t, project, ".vbx/config.yaml", "version: 1\nmounts:\n  - source: '"+two+"'\n    target: /workspaces/extra\n")
	c, err := Resolve(project, "/workspaces/project", global)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Mounts) != 1 || c.Mounts[0].Source != two || c.Mounts[0].Readonly {
		t.Fatal(c.Mounts)
	}
}
