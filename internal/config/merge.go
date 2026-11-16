package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"vbx"
)

func (c *Config) apply(p Config) {
	for key := range p.fields {
		switch key {
		case "resources.cpus":
			c.CPUs = p.CPUs
		case "resources.memory":
			c.Memory = p.Memory
		case "resources.disk":
			c.Disk = p.Disk
		case "launch.default":
			c.Default = p.Default
		case "launch.folder":
			c.Folder = p.Folder
			c.folderSet = true
		}
	}
	for _, m := range p.Mounts {
		found := false
		for i, v := range c.Mounts {
			if v.Target == m.Target {
				c.Mounts[i] = m
				found = true
				break
			}
		}
		if !found {
			c.Mounts = append(c.Mounts, m)
		}
	}
	c.Persistence = mergePaths(c.Persistence, p.Persistence)
	c.Modules = Unique(append(c.Modules, p.Modules...))
	c.APT = Unique(append(c.APT, p.APT...))
	c.EnvFiles = Unique(append(c.EnvFiles, p.EnvFiles...))
}
func source(read func(string) ([]byte, error), root, base string) (Config, error) {
	var c Config
	for _, name := range []string{"persistence.yaml", "config.yaml"} {
		file := path.Join(root, name)
		data, err := read(file)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return c, err
		}
		var p Config
		if err = p.readData(data, file, base, name == "config.yaml"); err != nil {
			return c, err
		}
		if name == "persistence.yaml" && len(p.Persistence) == 0 {
			return c, fmt.Errorf("%s requires nonempty persistence.paths", file)
		}
		c.apply(p)
		if c.fields == nil {
			c.fields = map[string]bool{}
		}
		for key := range p.fields {
			c.fields[key] = true
		}
	}
	return c, nil
}

// ModuleSettings reads embedded settings. Host paths must be absolute or home-relative.
func ModuleSettings(assets fs.FS, name string) (Config, error) {
	if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(name) {
		return Config{}, fmt.Errorf("invalid module identifier %q", name)
	}
	c, err := source(func(p string) ([]byte, error) { return fs.ReadFile(assets, p) }, "modules/"+name, "")
	if err != nil {
		return c, err
	}
	file := "modules/" + name + "/apt.txt"
	data, err := fs.ReadFile(assets, file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return c, err
	}
	var requirements []string
	for line, s := range strings.Split(string(data), "\n") {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		requirement, err := APT(s)
		if err != nil {
			return c, fmt.Errorf("%s:%d: %w", file, line+1, err)
		}
		requirements = append(requirements, requirement)
	}
	c.APT = Unique(append(requirements, c.APT...))
	for i := range c.Persistence {
		c.Persistence[i].Module = name
	}
	return c, nil
}
func bundled(name string) bool {
	switch name {
	case "aws-cli", "dev-tools", "node", "pi", "pi-web-fetch", "pi-project-persistent-sessions", "vscode-project-persistent-state":
		return true
	}
	return false
}
func Resolve(project, primary, local string) (Config, error) {
	return ResolveWithAssets(project, primary, local, vbx.Assets)
}

// ResolveWithAssets applies defaults, global modules, global settings, project modules, project settings.
func ResolveWithAssets(project, primary, local string, assets fs.FS) (Config, error) {
	global, err := source(os.ReadFile, local, local)
	if err != nil {
		return Config{}, err
	}
	projectSettings, err := source(os.ReadFile, filepath.Join(project, ".vbx"), project)
	if err != nil {
		return Config{}, err
	}
	c := Defaults()
	modules := func(names []string) error {
		seen := map[string]bool{}
		queue := append([]string{}, names...)
		for len(queue) > 0 {
			name := queue[0]
			queue = queue[1:]
			if seen[name] {
				continue
			}
			seen[name] = true
			if !bundled(name) {
				return fmt.Errorf("unknown bundled module %q", name)
			}
			p, err := ModuleSettings(assets, name)
			if err != nil {
				return err
			}
			queue = append(queue, p.Modules...)
			p.Modules = nil
			c.apply(p)
			c.Modules = Unique(append(c.Modules, name))
		}
		return nil
	}
	if err = modules(append(append([]string{}, c.Modules...), global.Modules...)); err != nil {
		return c, err
	}
	c.apply(global)
	if err = modules(projectSettings.Modules); err != nil {
		return c, err
	}
	c.apply(projectSettings)
	c.fields = nil
	return validateResolved(c, project, primary)
}
