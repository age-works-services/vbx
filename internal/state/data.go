package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"
	"vbx/internal/config"
	"vbx/internal/module"
)

var projectID = regexp.MustCompile(`^vbx-[a-z0-9][a-z0-9-]{0,23}-[0-9a-f]{12}$`)

func ValidProjectID(s string) bool { return projectID.MatchString(s) }

type Binding struct {
	Module, Host, Guest, Source string
	Writable                    bool
}
type Mapping struct{ Module, Guest, Data string }
type Record struct {
	Version  int
	Config   config.Config
	Bindings []Binding
	Mappings []Mapping
}

func EncodeRecord(r Record) ([]byte, error) {
	if _, err := Encode(r.Config); err != nil {
		return nil, err
	}
	r.Version = 1
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r)
}
func DecodeRecord(data []byte) (Record, error) {
	var r Record
	c, err := Decode(data)
	if err != nil {
		return r, err
	}
	if err = json.Unmarshal(data, &r); err != nil {
		return r, err
	}
	r.Config = c
	return r, r.Validate()
}
func (r Record) Validate() error {
	configData := false
	for _, p := range r.Config.Persistence {
		if p.Module == "" {
			configData = true
		}
	}
	count := len(r.Config.Modules)
	if configData {
		count++
	}
	if len(r.Bindings) != count {
		return fmt.Errorf("snapshot bindings do not match effective settings")
	}
	bound := map[string]bool{}
	for i, b := range r.Bindings {
		owner := ""
		guest := "/mnt/vbx-config-data"
		if i < len(r.Config.Modules) {
			owner = r.Config.Modules[i]
			guest = "/mnt/vbx-data/" + owner
			if !module.ValidName(owner) {
				return fmt.Errorf("invalid module identifier")
			}
		}
		if bound[b.Module] || b.Module != owner || !utf8.ValidString(b.Host) || strings.ContainsFunc(b.Host, unicode.IsControl) || !filepath.IsAbs(b.Host) || filepath.Clean(b.Host) != b.Host || b.Guest != guest || !b.Writable || b.Source != fmt.Sprintf("mount%d", 1+len(r.Config.Mounts)+i) {
			return fmt.Errorf("invalid captured data binding")
		}
		bound[b.Module] = true
	}
	if err := config.ValidatePersistence(r.Config.Persistence); err != nil {
		return err
	}
	var expected []Mapping
	for _, p := range r.Config.Persistence {
		if !bound[p.Module] {
			return fmt.Errorf("mapping has no data binding")
		}
		expected = append(expected, Mapping{Module: p.Module, Guest: p.Guest, Data: p.Data})
	}
	if !slices.Equal(expected, r.Mappings) {
		return fmt.Errorf("snapshot mappings differ from effective settings")
	}
	return nil
}
func (Paths) DataDir() (string, error) {
	base, err := Base("XDG_DATA_HOME", ".local/share")
	if err != nil {
		return "", err
	}
	if !utf8.ValidString(base) || strings.ContainsFunc(base, unicode.IsControl) {
		return "", fmt.Errorf("data-home path must be valid UTF-8 without control characters")
	}
	return base, nil
}
func privateDir(path string, create bool) error {
	st, err := os.Lstat(path)
	if os.IsNotExist(err) && create {
		if err = os.Mkdir(path, 0700); err != nil {
			return err
		}
		st, err = os.Lstat(path)
	}
	if err != nil {
		return err
	}
	stat, ok := st.Sys().(*syscall.Stat_t)
	if !st.IsDir() || !ok || int(stat.Uid) != os.Getuid() || st.Mode().Perm() != 0700 {
		return fmt.Errorf("module data path must be a real, owned mode-0700 directory: %s", path)
	}
	return nil
}
func (p Paths) ProjectData(id string, create bool) (string, error) {
	if !ValidProjectID(id) {
		return "", fmt.Errorf("invalid project identifier")
	}
	root, err := p.DataDir()
	if err != nil {
		return "", err
	}
	base := filepath.Dir(root)
	if create {
		if err = os.MkdirAll(base, 0700); err != nil {
			return "", err
		}
	}
	base, err = filepath.EvalSymlinks(base)
	if err != nil {
		return "", err
	}
	if strings.ContainsFunc(base, unicode.IsControl) {
		return "", fmt.Errorf("data-home path contains control characters")
	}
	root = filepath.Join(base, "vbx")
	for _, part := range []string{"", "projects", id, "modules"} {
		root = filepath.Join(root, part)
		if err = privateDir(root, create); err != nil {
			return "", err
		}
	}
	return root, nil
}
func (p Paths) AllocateData(id string, c config.Config, defs []module.Definition, project string) (Record, error) {
	r := Record{Config: c, Version: 1}
	if !ValidProjectID(id) {
		return r, fmt.Errorf("invalid project identifier")
	}
	if len(defs) != len(c.Modules) {
		return r, fmt.Errorf("module definitions differ from selection")
	}
	selected := map[string]bool{"": true}
	for i, d := range defs {
		if !module.ValidName(d.Name) || d.Name != c.Modules[i] || selected[d.Name] {
			return r, fmt.Errorf("invalid module selection")
		}
		selected[d.Name] = true
	}
	if err := config.ValidatePersistence(c.Persistence); err != nil {
		return r, err
	}
	for _, p := range c.Persistence {
		if !selected[p.Module] {
			return r, fmt.Errorf("persistence mapping has no selected storage owner")
		}
	}
	configData := false
	for _, p := range c.Persistence {
		if p.Module == "" {
			configData = true
		}
		r.Mappings = append(r.Mappings, Mapping{Module: p.Module, Guest: p.Guest, Data: p.Data})
	}
	if len(defs) == 0 && !configData {
		return r, r.Validate()
	}
	root, err := p.ProjectData(id, true)
	if err != nil {
		return r, err
	}
	sources := []string{project}
	for _, m := range c.Mounts {
		sources = append(sources, m.Source)
	}
	allocate := func(owner, host, guest string) error {
		for _, s := range sources {
			if module.Overlap(host, s) {
				return fmt.Errorf("persistent data overlaps host mount %s", s)
			}
		}
		if err := privateDir(host, true); err != nil {
			return err
		}
		r.Bindings = append(r.Bindings, Binding{Module: owner, Host: host, Guest: guest, Source: fmt.Sprintf("mount%d", 1+len(c.Mounts)+len(r.Bindings)), Writable: true})
		return nil
	}
	for _, d := range defs {
		if err = allocate(d.Name, filepath.Join(root, d.Name), "/mnt/vbx-data/"+d.Name); err != nil {
			return r, err
		}
	}
	if configData {
		if err = allocate("", filepath.Join(filepath.Dir(root), "config"), "/mnt/vbx-config-data"); err != nil {
			return r, err
		}
	}
	return r, r.Validate()
}
func ValidateData(id string, r Record) error {
	if !ValidProjectID(id) {
		return fmt.Errorf("invalid project identifier")
	}
	if err := r.Validate(); err != nil {
		return err
	}
	for _, b := range r.Bindings {
		parts := []string{"", "projects", id, "modules", b.Module}
		suffix := filepath.Join("projects", id, "modules", b.Module)
		if b.Module == "" {
			suffix = filepath.Join("projects", id, "config")
			parts = []string{"", "projects", id, "config"}
		}
		if !strings.HasSuffix(b.Host, string(filepath.Separator)+suffix) {
			return fmt.Errorf("invalid captured module data path")
		}
		root := strings.TrimSuffix(b.Host, string(filepath.Separator)+suffix)
		if filepath.Base(root) != "vbx" {
			return fmt.Errorf("invalid captured data root")
		}
		canonical, err := filepath.EvalSymlinks(filepath.Dir(root))
		if err != nil {
			return err
		}
		if canonical != filepath.Dir(root) {
			return fmt.Errorf("captured data root changed")
		}
		for _, part := range parts {
			root = filepath.Join(root, part)
			if err = privateDir(root, false); err != nil {
				return err
			}
		}
	}
	return nil
}
