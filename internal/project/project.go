package project

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Project struct{ Path, Name, Primary string }

var punctuation = regexp.MustCompile(`[^a-z0-9]+`)

func Resolve(path string) (Project, error) {
	var p Project
	absolute, err := filepath.Abs(path)
	if err != nil {
		return p, err
	}
	path, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return p, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return p, err
	}
	if !st.IsDir() || path == "/" || strings.ContainsAny(path, "\r\n") {
		return p, fmt.Errorf("select an existing project directory other than /, without line breaks")
	}
	base := []byte(filepath.Base(path))
	for i, c := range base {
		if c >= 'A' && c <= 'Z' {
			base[i] += 'a' - 'A'
		}
	}
	slug := strings.Trim(punctuation.ReplaceAllString(string(base), "-"), "-")
	if len(slug) > 24 {
		slug = slug[:24]
	}
	if slug == "" {
		slug = "project"
	}
	digest := sha256.Sum256([]byte(path))
	return Project{path, fmt.Sprintf("vbx-%s-%x", slug, digest[:6]), "/workspaces/" + slug}, nil
}
