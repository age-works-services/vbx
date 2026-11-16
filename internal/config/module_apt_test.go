package config

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestModuleAPTParticipatesInSettingsMerge(t *testing.T) {
	assets := fstest.MapFS{
		"modules/node/apt.txt":     {Data: []byte("# module requirements\n curl \njq   (>= 1.7)\n")},
		"modules/node/config.yaml": {Data: []byte("version: 1\napt:\n  packages:\n    - curl\n    - git\n")},
		"modules/pi/apt.txt":       {Data: []byte("unzip\ncurl\n")},
		"modules/pi/config.yaml":   {Data: []byte("version: 1\napt:\n  packages:\n    - python3\n")},
	}
	local, project := t.TempDir(), t.TempDir()
	putSettings(t, local, "config.yaml", "version: 1\nmodules:\n  - node\napt:\n  packages:\n    - git\n    - 'jq (<= 2)'\n")
	putSettings(t, project, ".vbx/config.yaml", "version: 1\nmodules:\n  - node\n  - pi\napt:\n  packages:\n    - 'jq (>= 1.7)'\n    - make\n")
	c, err := ResolveWithAssets(project, "/workspaces/project", local, assets)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"curl", "jq (>= 1.7)", "git", "jq (<= 2)", "unzip", "python3", "make"}
	if !reflect.DeepEqual(c.APT, want) {
		t.Fatalf("got %v, want %v", c.APT, want)
	}
	output := string(Format(c))
	for _, requirement := range want {
		encoded, _ := json.Marshal(requirement)
		if !strings.Contains(output, string(encoded)) {
			t.Fatal(output)
		}
	}
}

func TestInvalidModuleAPTStopsResolution(t *testing.T) {
	for _, requirement := range []string{"--allow-unauthenticated", "git | bash", "https://example.invalid/tool.deb"} {
		assets := fstest.MapFS{"modules/node/apt.txt": {Data: []byte("# comment\n" + requirement + "\n")}}
		local := t.TempDir()
		putSettings(t, local, "config.yaml", "version: 1\nmodules:\n  - node\n")
		_, err := ResolveWithAssets(t.TempDir(), "/workspaces/project", local, assets)
		if err == nil || !strings.Contains(err.Error(), "modules/node/apt.txt:2:") {
			t.Fatal("invalid module requirements accepted", err)
		}
	}
}
