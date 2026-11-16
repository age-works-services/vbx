package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestAllSettingsUseLayerOrder(t *testing.T) {
	global, project := t.TempDir(), t.TempDir()
	dirs := []string{t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()}
	envs := []string{}
	for i := 0; i < 4; i++ {
		name := fmt.Sprintf("env%d", i)
		putSettings(t, global, name, "VALUE=value\n")
		envs = append(envs, filepath.Join(global, name))
	}
	mount := func(source string, readonly bool) string {
		return fmt.Sprintf("mounts:\n  - source: %q\n    target: /workspaces/extra\n    readonly: %t\n", source, readonly)
	}
	env := func(index int) string { return fmt.Sprintf("envFiles:\n  - %q\n", envs[index]) }
	assets := fstest.MapFS{
		"modules/node/config.yaml": {Data: []byte("version: 1\nresources:\n  cpus: 3\n  memory: 5GiB\nlaunch:\n  default: terminal\napt:\n  packages:\n    - jq\n" + mount(dirs[0], true) + env(0))},
		"modules/pi/config.yaml":   {Data: []byte("version: 1\nresources:\n  cpus: 4\napt:\n  packages:\n    - curl\n" + mount(dirs[2], true) + env(2))},
	}
	putSettings(t, global, "config.yaml", "version: 1\nmodules:\n  - node\nlaunch:\n  default: vscode\napt:\n  packages:\n    - git\n"+mount(dirs[1], false)+env(1))
	putSettings(t, project, ".vbx/config.yaml", "version: 1\nmodules:\n  - pi\nresources:\n  memory: 9GiB\nlaunch:\n  folder: /workspaces/custom\napt:\n  packages:\n    - bash\n"+mount(dirs[3], false)+env(3))
	c, err := ResolveWithAssets(project, "/workspaces/project", global, assets)
	if err != nil {
		t.Fatal(err)
	}
	if c.CPUs != "4" || c.Memory != "9GiB" || c.Default != "vscode" || c.Folder != "/workspaces/custom" {
		t.Fatal(c)
	}
	if len(c.Mounts) != 1 || c.Mounts[0].Source != dirs[3] || c.Mounts[0].Readonly {
		t.Fatal(c.Mounts)
	}
	if !reflect.DeepEqual(c.EnvFiles, envs) || !reflect.DeepEqual(c.APT, []string{"jq", "git", "curl", "bash"}) || !reflect.DeepEqual(c.Modules, []string{"node", "pi"}) {
		t.Fatal(c)
	}
}
func TestModuleSidecarAndSettings(t *testing.T) {
	assets := fstest.MapFS{
		"modules/node/persistence.yaml": {Data: []byte("persistence:\n  paths:\n    - guest: ~/.tool\n      data: sidecar\n")},
		"modules/node/config.yaml":      {Data: []byte("version: 1\nmodules:\n  - pi\npersistence:\n  paths:\n    - guest: ~/.tool\n      data: inline\n")},
		"modules/pi/config.yaml":        {Data: []byte("version: 1\nresources:\n  memory: 8GiB\n")},
	}
	global, project := t.TempDir(), t.TempDir()
	putSettings(t, global, "config.yaml", "version: 1\nmodules:\n  - node\n")
	c, err := ResolveWithAssets(project, "/workspaces/project", global, assets)
	if err != nil {
		t.Fatal(err)
	}
	if c.Memory != "8GiB" || len(c.Persistence) != 1 || c.Persistence[0].Data != "inline" || c.Persistence[0].Module != "node" || !reflect.DeepEqual(c.Modules, []string{"node", "pi"}) {
		t.Fatal(c)
	}
}
func TestPersistenceFileValidation(t *testing.T) {
	for _, data := range []string{
		"paths:\n  - guest: ~/.tool\n    data: tool\n",
		"persistence:\n  paths:\n    - guest: ~/.tool\n      data: tool\n      unknown: x\n",
		"persistence:\n  paths:\n    - guest: ~/.tool\n      data: tool\n    - guest: ~/.tool\n      data: other\n",
		"persistence:\n  paths:\n    - guest: ~/.tool\n      data: tool\nresources:\n  cpus: 2\n",
	} {
		if _, err := ParsePersistence([]byte(data)); err == nil {
			t.Fatal("invalid sidecar accepted", data)
		}
	}
}
