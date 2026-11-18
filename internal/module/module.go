// Package module validates bundled module definitions and persistence declarations.
package module

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"vbx/internal/config"
)

type Path = config.PersistencePath
type Definition struct {
	Name  string
	Setup bool
	Paths []Path
}

func ValidName(s string) bool           { return config.ValidModuleName(s) }
func Relative(s string) bool            { return config.Relative(s) }
func Overlap(a, b string) bool          { return config.Overlap(a, b) }
func Validate(p Path) error             { return config.ValidatePath(p) }
func Parse(data []byte) ([]Path, error) { return config.ParsePersistence(data) }
func Load(assets fs.FS, names []string) ([]Definition, error) {
	var defs []Definition
	for _, name := range names {
		if !ValidName(name) {
			return nil, fmt.Errorf("invalid module identifier %q", name)
		}
		root := path.Join("modules", name)
		d := Definition{Name: name}
		if st, err := fs.Stat(assets, root+"/setup.sh"); err == nil {
			if !st.Mode().IsRegular() {
				return nil, fmt.Errorf("%s/setup.sh must be regular", root)
			}
			d.Setup = true
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		c, err := config.ModuleSettings(assets, name)
		if err != nil {
			return nil, err
		}
		d.Paths = c.Persistence
		if !d.Setup && len(d.Paths) == 0 {
			if _, err := fs.Stat(assets, root+"/config.yaml"); err != nil {
				return nil, fmt.Errorf("%s requires settings or setup.sh", root)
			}
		}
		defs = append(defs, d)
	}
	return defs, nil
}

// CheckOverlay validates a captured entry using its own type, including symlinks.
func CheckOverlay(rel string, isDir bool, defs []Definition, kinds map[string]bool) error {
	if wasDir, ok := kinds[rel]; ok && !wasDir && isDir {
		return fmt.Errorf("home overlay replaces a non-directory ancestor: %s", rel)
	}
	guest := "~/" + rel
	for ancestor := path.Dir(rel); ancestor != "."; ancestor = path.Dir(ancestor) {
		if dir, ok := kinds[ancestor]; ok && !dir {
			return fmt.Errorf("home overlay traverses non-directory %s", ancestor)
		}
	}
	protected := []string{"~/.config/vbx/environment.sh", "~/.local/state/vbx"}
	for _, d := range defs {
		for _, p := range d.Paths {
			protected = append(protected, p.Guest)
		}
	}
	for _, p := range protected {
		if guest == p || strings.HasPrefix(guest, p+"/") || (!isDir && strings.HasPrefix(p, guest+"/")) {
			return fmt.Errorf("home overlay %s conflicts with managed path %s", rel, p)
		}
	}
	kinds[rel] = isDir
	return nil
}
