package provision

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vbx"
	"vbx/internal/config"
)

func TestEmptyLists(t *testing.T) {
	var b bytes.Buffer
	if err := Build(&b, vbx.Assets, bundleRecord(config.Defaults()), t.TempDir(), t.TempDir()); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	r := tar.NewReader(&b)
	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Name == "apt.txt" || h.Name == "modules.list" {
			seen[h.Name] = true
			data, err := io.ReadAll(r)
			if err != nil || len(data) != 0 {
				t.Fatalf("%s = %q, %v", h.Name, data, err)
			}
		}
	}
	if len(seen) != 2 {
		t.Fatal(seen)
	}
}

func TestDanglingInputs(t *testing.T) {
	for _, kind := range []string{"local-home", "project-home", "setup"} {
		t.Run(kind, func(t *testing.T) {
			local, p := t.TempDir(), t.TempDir()
			os.Mkdir(filepath.Join(p, ".vbx"), 0700)
			path := filepath.Join(local, "home")
			if kind == "project-home" {
				path = filepath.Join(p, ".vbx/home")
			}
			if kind == "setup" {
				path = filepath.Join(p, ".vbx/setup.sh")
			}
			if err := os.Symlink(filepath.Join(p, "missing"), path); err != nil {
				t.Fatal(err)
			}
			if err := Build(io.Discard, vbx.Assets, bundleRecord(config.Defaults()), local, p); err == nil {
				t.Fatal("dangling input skipped")
			}
		})
	}
}
func TestTopLevelInputSymlinks(t *testing.T) {
	local, p, target := t.TempDir(), t.TempDir(), t.TempDir()
	os.Mkdir(filepath.Join(p, ".vbx"), 0700)
	os.WriteFile(filepath.Join(target, "file"), []byte("home"), 0600)
	os.Symlink(target, filepath.Join(local, "home"))
	setup := filepath.Join(target, "setup")
	os.WriteFile(setup, []byte("echo setup"), 0600)
	os.Symlink(setup, filepath.Join(p, ".vbx/setup.sh"))
	var b bytes.Buffer
	if err := Build(&b, vbx.Assets, bundleRecord(config.Defaults()), local, p); err != nil {
		t.Fatal(err)
	}
	r := tar.NewReader(&b)
	found := map[string]bool{}
	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		found[h.Name] = true
	}
	if !found["local-home/file"] || !found["project-setup.sh"] {
		t.Fatal(found)
	}
}
func TestExportWhitespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	os.WriteFile(path, []byte("export  KEY=value\nexport\tOTHER=second\nexported=literal\n"), 0600)
	out, err := Environment([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"export KEY='value'", "export OTHER='second'", "export exported='literal'"} {
		if !strings.Contains(out, line) {
			t.Fatal(out)
		}
	}
}
