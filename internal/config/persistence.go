package config

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// PersistencePath retains the winning declaration's storage owner. Empty Module means configuration-owned data.
type PersistencePath struct{ Guest, Data, Module string }

func Relative(s string) bool {
	if !utf8.ValidString(s) || s == "" || strings.HasPrefix(s, "/") || strings.ContainsAny(s, "$`\\") || strings.ContainsFunc(s, unicode.IsControl) {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if p == "" || p == "." || p == ".." {
			return false
		}
	}
	return true
}
func Overlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}
func ValidatePath(p PersistencePath) error {
	if !strings.HasPrefix(p.Guest, "~/") || !Relative(strings.TrimPrefix(p.Guest, "~/")) || !Relative(p.Data) {
		return fmt.Errorf("persistence paths require guest: ~/DIRECTORY and data: RELATIVE_DIRECTORY")
	}
	for _, r := range []string{"~/.config/vbx", "~/.local/state/vbx"} {
		if Overlap(p.Guest, r) {
			return fmt.Errorf("persistence path %s overlaps %s", p.Guest, r)
		}
	}
	return nil
}
func mergePaths(old, next []PersistencePath) []PersistencePath {
	for _, p := range next {
		found := false
		for i, v := range old {
			if v.Guest == p.Guest {
				old[i] = p
				found = true
				break
			}
		}
		if !found {
			old = append(old, p)
		}
	}
	return old
}
func ValidatePersistence(paths []PersistencePath) error {
	for i, p := range paths {
		if err := ValidatePath(p); err != nil {
			return err
		}
		for _, q := range paths[:i] {
			if Overlap(p.Guest, q.Guest) {
				return fmt.Errorf("overlapping persistence guest paths %s and %s", p.Guest, q.Guest)
			}
			if p.Module == q.Module && Overlap(p.Data, q.Data) {
				return fmt.Errorf("overlapping persistence data paths %s and %s", p.Data, q.Data)
			}
		}
	}
	return nil
}

// ParsePersistence accepts only the persistence.paths subset of the configuration schema.
func ParsePersistence(data []byte) ([]PersistencePath, error) {
	c := Config{}
	if err := c.readData(data, "persistence.yaml", "", false); err != nil {
		return nil, err
	}
	if len(c.Persistence) == 0 {
		return nil, fmt.Errorf("persistence.yaml requires nonempty persistence.paths")
	}
	return c.Persistence, nil
}
