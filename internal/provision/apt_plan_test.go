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

func TestBundleUsesResolvedAPTPlan(t *testing.T) {
	assets := fstest.MapFS{
		"modules/node/setup.sh":    {Data: []byte("true\n")},
		"modules/node/apt.txt":     {Data: []byte("curl\njq (>= 1.7)\n")},
		"modules/node/config.yaml": {Data: []byte("version: 1\napt:\n  packages:\n    - git\n")},
	}
	for _, name := range []string{"provision-guest.sh", "apt-install.sh", "guest-shell-system.sh", "npm-policy.sh", "persistence.sh"} {
		data, err := vbx.Assets.ReadFile("scripts/" + name)
		if err != nil {
			t.Fatal(err)
		}
		assets["scripts/"+name] = &fstest.MapFile{Data: data}
	}
	local, project := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(local, "config.yaml"), []byte("version: 1\nmodules:\n  - node\napt:\n  packages:\n    - 'jq (<= 2)'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := config.ResolveWithAssets(project, "/workspaces/project", local, assets)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := state.EncodeRecord(bundleRecord(c))
	if err != nil {
		t.Fatal(err)
	}
	record, err := state.DecodeRecord(saved)
	if err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	if err = Build(&archive, assets, record, local, project); err != nil {
		t.Fatal(err)
	}
	reader := tar.NewReader(&archive)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			t.Fatal("missing captured APT plan")
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Name != "apt.txt" {
			continue
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		if want := strings.Join(c.APT, "\n") + "\n"; string(data) != want {
			t.Fatalf("captured %q, resolved %q", data, want)
		}
		break
	}
}
