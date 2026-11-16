package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestFormatDefaults(t *testing.T) {
	c := Defaults()
	c.Folder = "/workspaces/project"
	want := `version: 1
resources:
  cpus: 2
  memory: "4GiB"
  disk: "30GiB"
launch:
  default: "vscode"
  folder: "/workspaces/project"
mounts: []
modules: []
persistence:
  paths: []
apt:
  packages: []
envFiles: []
`
	if got := string(Format(c)); got != want {
		t.Fatalf("got:\n%swant:\n%s", got, want)
	}
}
func TestFormatRoundTrip(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "project")
	source := filepath.Join(root, `source 'quoted' "double"`)
	local := filepath.Join(root, "local")
	for _, dir := range []string{p, source, local} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	env := filepath.Join(local, "env file")
	if err := os.WriteFile(env, []byte("TOKEN=private-value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c := Config{CPUs: "4", Memory: "8GiB", Disk: "40GiB", Default: "terminal", Folder: "/workspaces", Mounts: []Mount{{Source: source, Target: "/workspaces/source", Readonly: true}}, Modules: []string{"dev-tools", "node"}, Persistence: []PersistencePath{{Guest: "~/.tool/data", Data: "tool-data"}}, APT: []string{"git", "jq (>= 1.7)"}, EnvFiles: []string{env}, folderSet: true}
	data := Format(c)
	if err := os.WriteFile(filepath.Join(local, "config.yaml"), data, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveWithAssets(p, "/workspaces/project", local, fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, c) {
		t.Fatalf("%#v != %#v", got, c)
	}
}
