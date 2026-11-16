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
)

func TestEnvironment(t *testing.T) {
	d := t.TempDir()
	a := filepath.Join(d, "a")
	b := filepath.Join(d, "b")
	os.WriteFile(a, []byte("ORDER=local\nSECRET=literal $() `command` ' quote\n"), 0600)
	os.WriteFile(b, []byte("ORDER=project\nEMPTY=\n"), 0600)
	out, e := Environment([]string{a, b})
	if e != nil || !strings.Contains(out, "export ORDER='project'") || !strings.Contains(out, `'"'"'`) {
		t.Fatal(out, e)
	}
	for _, s := range []string{"PATH=bad", "VBX_INTERNAL=bad", "XDG_CONFIG_HOME=bad", "bad syntax", "TOKEN=\"secret", "TOKEN=\"bad\\quote\""} {
		os.WriteFile(a, []byte(s), 0600)
		_, e := Environment([]string{a})
		if e == nil || strings.Contains(e.Error(), "secret") {
			t.Fatal(s, e)
		}
	}
}
func TestBundle(t *testing.T) {
	d := t.TempDir()
	local := filepath.Join(d, "local")
	p := filepath.Join(d, "project")
	os.MkdirAll(filepath.Join(local, "home/.hidden"), 0700)
	os.MkdirAll(filepath.Join(p, ".vbx/home"), 0700)
	f := filepath.Join(local, "home/.hidden/readonly")
	os.WriteFile(f, []byte("private fixture"), 0400)
	os.Symlink("/absolute/target", filepath.Join(p, ".vbx/home/link"))
	os.WriteFile(filepath.Join(p, ".vbx/setup.sh"), []byte("echo setup"), 0600)
	c := config.Defaults()
	c.Modules = []string{"dev-tools"}
	var b bytes.Buffer
	if e := Build(&b, vbx.Assets, bundleRecord(c), local, p); e != nil {
		t.Fatal(e)
	}
	seen := map[string]*tar.Header{}
	tr := tar.NewReader(&b)
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			t.Fatal(e)
		}
		seen[h.Name] = h
	}
	for _, n := range []string{"modules/dev-tools/setup.sh", "provision.sh", "apt.txt", "environment.sh", "project-setup.sh", "local-home/.hidden/readonly", "project-home/link"} {
		if seen[n] == nil {
			t.Fatal("missing", n)
		}
	}
	if seen["local-home/.hidden/readonly"].Mode != 0400 || seen["project-home/link"].Linkname != "/absolute/target" {
		t.Fatal("metadata lost")
	}
}
func TestInvalidAsset(t *testing.T) {
	assets := fstest.MapFS{}
	for _, n := range []string{"scripts/provision-guest.sh", "scripts/apt-install.sh", "scripts/guest-shell-system.sh", "scripts/npm-policy.sh", "scripts/persistence.sh", "modules/pi/setup.sh"} {
		assets[n] = &fstest.MapFile{Data: []byte("fixture")}
	}
	assets["modules/pi/apt.txt"] = &fstest.MapFile{Data: []byte("--allow-unauthenticated")}
	c := config.Defaults()
	c.Modules = []string{"pi"}
	if e := Build(io.Discard, assets, bundleRecord(c), t.TempDir(), t.TempDir()); e == nil {
		t.Fatal("accepted unsafe apt manifest")
	}
}
