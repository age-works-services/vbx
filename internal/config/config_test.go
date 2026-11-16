package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLayers(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, "local")
	p := filepath.Join(root, "project")
	os.MkdirAll(local, 0700)
	os.MkdirAll(filepath.Join(p, ".vbx"), 0700)
	os.WriteFile(filepath.Join(local, "config.yaml"), []byte("version: 1\nresources:\n  cpus: 3\nmodules:\n  - pi\n  - node\napt:\n  packages:\n    - git\n    - 'nodejs (>= 22.19)'\n"), 0600)
	os.WriteFile(filepath.Join(p, ".vbx/config.yaml"), []byte("version: 1\nresources:\n  cpus: 4\nmodules:\n  - pi\n  - pi-web-fetch\napt:\n  packages:\n    - git\n    - 'nodejs (<= 30)'\n"), 0600)
	c, e := ResolveWithAssets(p, "/workspaces/project", local, fstest.MapFS{})
	if e != nil {
		t.Fatal(e)
	}
	if c.CPUs != "4" || !reflect.DeepEqual(c.Modules, []string{"pi", "node", "pi-web-fetch"}) || !reflect.DeepEqual(c.APT, []string{"git", "nodejs (>= 22.19)", "nodejs (<= 30)"}) {
		t.Fatalf("%#v", c)
	}
}
func TestInvalid(t *testing.T) {
	for _, text := range []string{"version: 2", "version: 1\nunknown: true", "version: 1\nversion: 1", "version: 1\nresources:\n  cpus: 0", "version: 1\nmodules:\n  - browser", "version: 1\nlaunch:\n  folder: relative", "version: 1\nmounts:\n  - source: /", "version: 1\napt:\n  packages:\n    - 'git | bash'", "version: 1\nresources:\n\tcpus: 3"} {
		t.Run(text, func(t *testing.T) {
			d := t.TempDir()
			os.WriteFile(filepath.Join(d, "config.yaml"), []byte(text+"\n"), 0600)
			if _, e := Resolve(t.TempDir(), "/workspaces/app", d); e == nil {
				t.Fatal("accepted invalid config")
			}
		})
	}
}
func TestAPT(t *testing.T) {
	for _, s := range []string{"https://example.invalid/a.deb", "--flag", "git;touch /tmp/pwned", "git | bash", "git (= latest)", "git, sudo", "Conflicts: git"} {
		if _, e := APT(s); e == nil {
			t.Fatalf("accepted %s", s)
		}
	}
	v, e := APT(" jq   (>= 1.7) ")
	if e != nil || v != "jq (>= 1.7)" {
		t.Fatal(v, e)
	}
}
func TestScalar(t *testing.T) {
	for _, s := range []string{"'a ''quoted'' path'", "\"literal $(touch /tmp/pwned)\"", "plain # comment"} {
		if _, e := Scalar(s); e != nil {
			t.Fatal(e)
		}
	}
	for _, s := range []string{"'unterminated", "\"a\\n\"", "&anchor", "[x]"} {
		if _, e := Scalar(s); e == nil {
			t.Fatalf("accepted %s", s)
		}
	}
}
func TestMountsAndEnv(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "project")
	os.MkdirAll(filepath.Join(p, ".vbx"), 0700)
	os.Mkdir(filepath.Join(d, "api"), 0700)
	os.WriteFile(filepath.Join(d, "local.env"), []byte("A=b\n"), 0600)
	text := "version: 1\nmounts:\n  - source: api\n    target: /workspaces/api\n    readonly: true\nenvFiles:\n  - local.env\n"
	os.WriteFile(filepath.Join(d, "config.yaml"), []byte(text), 0600)
	c, e := Resolve(p, "/workspaces/project", d)
	if e != nil || len(c.Mounts) != 1 || !c.Mounts[0].Readonly || c.Folder != "/workspaces" {
		t.Fatal(c, e)
	}
	os.WriteFile(filepath.Join(d, "config.yaml"), []byte(strings.ReplaceAll(text, "source: api", "source: project")), 0600)
	if _, e = Resolve(p, "/workspaces/project", d); e == nil {
		t.Fatal("accepted overlapping mount")
	}
}
