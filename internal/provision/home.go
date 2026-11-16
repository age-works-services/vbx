package provision

import (
	"fmt"
	"os"
	"path/filepath"
	"vbx/internal/config"
	"vbx/internal/module"
)

// HomeEntry describes metadata only. Path is the capture source; Relative is beneath home.
// A missing overlay contributes a root entry with nil Info for archive construction.
type HomeEntry struct {
	Layer, Relative, Path string
	Info                  os.FileInfo
	Link                  string
}

func (e HomeEntry) ArchiveName() string {
	root := "local-home"
	if e.Layer == "project" {
		root = "project-home"
	}
	if e.Relative == "." {
		return root
	}
	return root + "/" + e.Relative
}

// WalkHome visits both layers in copy order, using the same collision checks for
// inspection and capture. It never reads regular-file contents or follows child links.
func WalkHome(local, project string, paths []config.PersistencePath, visit func(HomeEntry) error) error {
	kinds := map[string]bool{}
	defs := []module.Definition{{Paths: paths}}
	for _, layer := range []struct{ name, root string }{{"global", filepath.Join(local, "home")}, {"project", filepath.Join(project, ".vbx/home")}} {
		if _, err := os.Lstat(layer.root); os.IsNotExist(err) {
			if err = visit(HomeEntry{Layer: layer.name, Relative: "."}); err != nil {
				return err
			}
			continue
		} else if err != nil {
			return err
		}
		if st, err := os.Stat(layer.root); err != nil || !st.IsDir() {
			return fmt.Errorf("expected a home directory: %s", layer.root)
		}
		root, err := filepath.EvalSymlinks(layer.root)
		if err != nil {
			return err
		}
		err = filepath.Walk(root, func(file string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, file)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if rel != "." {
				if wasDir, ok := kinds[rel]; ok && wasDir != info.IsDir() {
					return fmt.Errorf("home overlay file/directory conflict: %s", rel)
				}
				if err = module.CheckOverlay(rel, info.IsDir(), defs, kinds); err != nil {
					return err
				}
			}
			entry := HomeEntry{Layer: layer.name, Relative: rel, Path: file, Info: info}
			if info.Mode()&os.ModeSymlink != 0 {
				entry.Link, err = os.Readlink(file)
				if err != nil {
					return err
				}
			} else if !info.Mode().IsRegular() && !info.IsDir() {
				return fmt.Errorf("unsupported home file: %s", file)
			}
			return visit(entry)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
