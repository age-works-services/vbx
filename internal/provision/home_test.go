package provision

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"vbx"
	"vbx/internal/config"
	"vbx/internal/state"
)

func TestHomeWalkPreservesMetadataWithoutFollowingLinks(t *testing.T) {
	local, project, target := t.TempDir(), t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "settings"), []byte("private"), 0400); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(local, "home")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing-target", filepath.Join(target, "link")); err != nil {
		t.Fatal(err)
	}
	seen := map[string]HomeEntry{}
	if err := WalkHome(local, project, nil, func(e HomeEntry) error { seen[e.ArchiveName()] = e; return nil }); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 4 || seen["local-home/settings"].Info.Mode().Perm() != 0400 || seen["local-home/link"].Link != "missing-target" || seen["project-home"].Info != nil {
		t.Fatal(seen)
	}
}
func TestHomeInspectionAndCaptureRejectSameConflicts(t *testing.T) {
	for _, scenario := range []string{"persistence", "directory to file", "file to directory"} {
		t.Run(scenario, func(t *testing.T) {
			local, project := t.TempDir(), t.TempDir()
			var paths []config.PersistencePath
			file := filepath.Join(local, "home/collision")
			if scenario == "directory to file" {
				file = filepath.Join(file, "child")
			}
			if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte("valuable"), 0600); err != nil {
				t.Fatal(err)
			}
			other := filepath.Join(project, ".vbx/home/collision")
			if scenario == "file to directory" {
				other = filepath.Join(other, "child")
			}
			if scenario == "persistence" {
				paths = []config.PersistencePath{{Guest: "~/collision", Data: "collision"}}
			} else {
				if err := os.MkdirAll(filepath.Dir(other), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(other, []byte("project"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			inspectErr := WalkHome(local, project, paths, func(HomeEntry) error { return nil })
			if inspectErr == nil {
				t.Fatal("inspection accepted conflict")
			}
			c := config.Defaults()
			c.Persistence = paths
			record := bundleRecord(c)
			if len(paths) > 0 {
				record.Bindings = []state.Binding{{Host: "/test-data/config", Guest: "/mnt/vbx-config-data", Source: "mount1", Writable: true}}
				record.Mappings = []state.Mapping{{Guest: paths[0].Guest, Data: paths[0].Data}}
			}
			err := Build(io.Discard, vbx.Assets, record, local, project)
			if err == nil || err.Error() != inspectErr.Error() {
				t.Fatal("inspection and capture disagree", inspectErr, err)
			}
			if data, err := os.ReadFile(file); err != nil || string(data) != "valuable" {
				t.Fatal("source changed", err)
			}
		})
	}
}
