// Package config implements VBX's restricted version-1 configuration grammar.
package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Mount struct {
	Source, Target string
	Readonly       bool
}
type Config struct {
	CPUs, Memory, Disk, Default, Folder string
	Mounts                              []Mount
	Modules, EnvFiles, APT              []string
	Persistence                         []PersistencePath
	fields                              map[string]bool
	folderSet                           bool
}

func Defaults() Config { return Config{CPUs: "2", Memory: "4GiB", Disk: "30GiB", Default: "vscode"} }
func badControl(s string) bool {
	for _, c := range s {
		if c < 32 || c == 127 {
			return true
		}
	}
	return false
}
func Scalar(s string) (string, error) {
	s = strings.TrimSpace(s)
	fail := func() (string, error) {
		return "", fmt.Errorf("expected a plain or quoted scalar without control characters")
	}
	if badControl(s) || s == "" {
		return fail()
	}
	if s[0] == '\'' {
		end := ""
		i := 1
		closed := false
		for i < len(s) {
			if s[i] == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					end += "'"
					i += 2
					continue
				}
				i++
				closed = true
				break
			}
			end += s[i : i+1]
			i++
		}
		tail := strings.TrimSpace(s[i:])
		if !closed || (tail != "" && !strings.HasPrefix(tail, "#")) {
			return fail()
		}
		return end, nil
	}
	if s[0] == '"' {
		for i := 1; i < len(s); i++ {
			if s[i] == '\\' {
				i++
				continue
			}
			if s[i] == '"' {
				var v string
				tail := strings.TrimSpace(s[i+1:])
				if json.Unmarshal([]byte(s[:i+1]), &v) != nil || badControl(v) || (tail != "" && !strings.HasPrefix(tail, "#")) {
					return fail()
				}
				return v, nil
			}
		}
		return fail()
	}
	s = stripComment(s)
	if s == "" || strings.ContainsAny(s, "'\"[]{}") || strings.Contains(s, ": ") || strings.ContainsAny(s[:1], "!&*|>@`") {
		return fail()
	}
	return s, nil
}
func stripComment(s string) string {
	for i := 1; i < len(s); i++ {
		if s[i] == '#' && (s[i-1] == ' ' || s[i-1] == '\t') {
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}

var aptRE = regexp.MustCompile(`^([a-z0-9][a-z0-9+.-]*(?::[a-z0-9][a-z0-9-]*)?)(?:\s+\((>=|<=|=)\s+([0-9][A-Za-z0-9.+:~\-]*)\))?$`)

func APT(s string) (string, error) {
	m := aptRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return "", fmt.Errorf("expected a package name or package (>=|<=|= VERSION)")
	}
	if m[2] != "" {
		return m[1] + " (" + m[2] + " " + m[3] + ")", nil
	}
	return m[1], nil
}
func Unique(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range values {
		if !seen[v] {
			out = append(out, v)
			seen[v] = true
		}
	}
	return out
}
func expand(path, base string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, e := os.UserHomeDir()
		if e != nil {
			return "", e
		}
		path = home + strings.TrimPrefix(path, "~")
	}
	if !filepath.IsAbs(path) {
		if base == "" {
			return "", fmt.Errorf("module host paths must be absolute or home-relative")
		}
		path = filepath.Join(base, path)
	}
	return filepath.EvalSymlinks(path)
}
func validateResolved(c Config, project, primary string) (Config, error) {
	n, e := strconv.Atoi(c.CPUs)
	if e != nil || n < 1 || !regexp.MustCompile(`^[1-9][0-9]*$`).MatchString(c.CPUs) {
		return c, fmt.Errorf("resources.cpus: expected a positive integer")
	}
	for _, v := range []string{c.Memory, c.Disk} {
		if !regexp.MustCompile(`^[1-9][0-9]*(MiB|GiB|TiB)$`).MatchString(v) {
			return c, fmt.Errorf("resources: use a positive integer with MiB, GiB, or TiB")
		}
	}
	if c.Default != "vscode" && c.Default != "terminal" && c.Default != "workspace-only" {
		return c, fmt.Errorf("launch.default: use vscode, terminal, or workspace-only")
	}
	if !c.folderSet {
		c.Folder = primary
		if len(c.Mounts) > 0 {
			c.Folder = "/workspaces"
		}
	}
	if !strings.HasPrefix(c.Folder, "/") || badControl(c.Folder) {
		return c, fmt.Errorf("launch.folder: expected an absolute guest path")
	}
	targets := map[string]bool{primary: true}
	sources := []string{project}
	for _, m := range c.Mounts {
		if !regexp.MustCompile(`^/workspaces/[a-zA-Z0-9_.-]+$`).MatchString(m.Target) || m.Target == "/workspaces/." || m.Target == "/workspaces/.." || targets[m.Target] {
			return c, fmt.Errorf("invalid or duplicate mount target %q", m.Target)
		}
		targets[m.Target] = true
		for _, s := range sources {
			if m.Source == s || strings.HasPrefix(m.Source, s+"/") || strings.HasPrefix(s, m.Source+"/") {
				return c, fmt.Errorf("overlapping host mount directories")
			}
		}
		sources = append(sources, m.Source)
	}
	c.Modules = Unique(c.Modules)
	for _, m := range c.Modules {
		if !bundled(m) {
			return c, fmt.Errorf("unknown bundled module %q", m)
		}
	}
	c.APT = Unique(c.APT)
	c.EnvFiles = Unique(c.EnvFiles)
	if err := ValidatePersistence(c.Persistence); err != nil {
		return c, err
	}
	return c, nil
}
func (c *Config) readData(data []byte, path, base string, version bool) error {
	seen := map[string]bool{}
	if c.fields == nil {
		c.fields = map[string]bool{}
	}
	section := ""
	item := ""
	index := -1
	line := 0
	scan := bufio.NewScanner(bytes.NewReader(data))
	scan.Buffer(make([]byte, 4096), 1024*1024)
	for scan.Scan() {
		line++
		raw := scan.Text()
		where := fmt.Sprintf("%s:%d", path, line)
		fail := func(msg string) error { return fmt.Errorf("%s: %s", where, msg) }
		if strings.Contains(raw, "\t") {
			return fail("tabs are unsupported")
		}
		if strings.TrimSpace(raw) == "" || strings.HasPrefix(strings.TrimSpace(raw), "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		text := strings.TrimSpace(raw)
		if indent == 0 {
			key, value, ok := strings.Cut(text, ":")
			if !ok || seen[key] {
				return fail("invalid or duplicate key")
			}
			seen[key] = true
			if !version && key != "persistence" {
				return fail("persistence.yaml supports only persistence.paths")
			}
			section = key
			item = ""
			index = -1
			if key == "version" {
				v, e := Scalar(value)
				if e != nil || v != "1" {
					return fail("version: 1 is required")
				}
				continue
			}
			if key != "resources" && key != "launch" && key != "mounts" && key != "modules" && key != "envFiles" && key != "apt" && key != "persistence" {
				return fail("unsupported key " + key)
			}
			v := strings.TrimSpace(strings.SplitN(value, "#", 2)[0])
			if v == "[]" && (key == "mounts" || key == "modules" || key == "envFiles") {
				section = ""
			} else if v != "" {
				return fail("expected a block mapping/list")
			}
			continue
		}
		if section == "resources" || section == "launch" {
			if indent != 2 {
				return fail("expected two-space indentation")
			}
			key, value, ok := strings.Cut(text, ":")
			if !ok || seen[section+"."+key] {
				return fail("invalid or duplicate key")
			}
			seen[section+"."+key] = true
			c.fields[section+"."+key] = true
			v, e := Scalar(value)
			if e != nil {
				return fail(e.Error())
			}
			switch section + "." + key {
			case "resources.cpus":
				c.CPUs = v
			case "resources.memory":
				c.Memory = v
			case "resources.disk":
				c.Disk = v
			case "launch.default":
				c.Default = v
			case "launch.folder":
				c.Folder = v
				c.folderSet = true
			default:
				return fail("unsupported key")
			}
			continue
		}
		if section == "persistence" {
			if indent == 2 {
				key, value, ok := strings.Cut(text, ":")
				v := strings.TrimSpace(strings.SplitN(value, "#", 2)[0])
				if !ok || key != "paths" || seen["persistence.paths"] || (v != "" && v != "[]") {
					return fail("expected persistence.paths list")
				}
				seen["persistence.paths"] = true
				item = "paths"
				if v == "[]" {
					item = ""
				}
				continue
			}
			if item != "paths" {
				return fail("expected persistence.paths")
			}
			entry := text
			if indent == 4 && strings.HasPrefix(text, "- ") {
				if index >= 0 && (!seen["path.guest"] || !seen["path.data"]) {
					return fail("persistence entry requires guest and data")
				}
				c.Persistence = append(c.Persistence, PersistencePath{})
				index = len(c.Persistence) - 1
				delete(seen, "path.guest")
				delete(seen, "path.data")
				entry = text[2:]
			} else if indent != 6 || index < 0 {
				return fail("invalid persistence entry")
			}
			key, value, ok := strings.Cut(entry, ":")
			if !ok || (key != "guest" && key != "data") || seen["path."+key] {
				return fail("invalid or duplicate persistence field")
			}
			seen["path."+key] = true
			v, err := Scalar(value)
			if err != nil {
				return fail(err.Error())
			}
			if key == "guest" {
				c.Persistence[index].Guest = v
			} else {
				c.Persistence[index].Data = v
			}
			continue
		}
		if section == "apt" && indent == 2 {
			key, value, ok := strings.Cut(text, ":")
			v := strings.TrimSpace(strings.SplitN(value, "#", 2)[0])
			if !ok || key != "packages" || seen["apt.packages"] || (v != "" && v != "[]") {
				return fail("expected apt.packages list")
			}
			seen["apt.packages"] = true
			item = "packages"
			if v == "[]" {
				item = ""
			}
			continue
		}
		if (section == "modules" || section == "envFiles") && indent == 2 && strings.HasPrefix(text, "- ") {
			v, e := Scalar(text[2:])
			if e != nil {
				return fail(e.Error())
			}
			if section == "modules" {
				c.Modules = append(c.Modules, v)
			} else {
				p, e := expand(v, base)
				if e != nil {
					return fail("environment file must exist")
				}
				st, e := os.Stat(p)
				if e != nil || !st.Mode().IsRegular() {
					return fail("environment file must be regular")
				}
				c.EnvFiles = append(c.EnvFiles, p)
			}
			continue
		}
		if section == "apt" && item == "packages" && indent == 4 && strings.HasPrefix(text, "- ") {
			v, e := Scalar(text[2:])
			if e != nil {
				return fail(e.Error())
			}
			v, e = APT(v)
			if e != nil {
				return fail(e.Error())
			}
			c.APT = append(c.APT, v)
			continue
		}
		if section == "mounts" {
			if indent == 2 && strings.HasPrefix(text, "- source:") {
				v, e := Scalar(strings.TrimPrefix(text, "- source:"))
				if e != nil {
					return fail(e.Error())
				}
				p, e := expand(v, base)
				if e != nil {
					return fail("mount source must exist")
				}
				st, e := os.Stat(p)
				if e != nil || !st.IsDir() || p == "/" || badControl(p) {
					return fail("invalid mount source")
				}
				c.Mounts = append(c.Mounts, Mount{Source: p})
				index = len(c.Mounts) - 1
				delete(seen, "mount.target")
				delete(seen, "mount.readonly")
				continue
			}
			if indent == 4 && index >= 0 {
				key, value, ok := strings.Cut(text, ":")
				if !ok || seen["mount."+key] {
					return fail("invalid or duplicate mount key")
				}
				seen["mount."+key] = true
				v, e := Scalar(value)
				if e != nil {
					return fail(e.Error())
				}
				switch key {
				case "target":
					c.Mounts[index].Target = v
				case "readonly":
					if v != "true" && v != "false" {
						return fail("readonly must be true or false")
					}
					c.Mounts[index].Readonly = v == "true"
				default:
					return fail("unsupported mount key")
				}
				continue
			}
		}
		return fail("unsupported configuration syntax")
	}
	if err := scan.Err(); err != nil {
		return err
	}
	if version && !seen["version"] {
		return fmt.Errorf("%s: version: 1 is required", path)
	}
	guests := map[string]bool{}
	for _, p := range c.Persistence {
		if err := ValidatePath(p); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if guests[p.Guest] {
			return fmt.Errorf("%s: duplicate persistence guest %s", path, p.Guest)
		}
		guests[p.Guest] = true
	}
	targets := map[string]bool{}
	for _, m := range c.Mounts {
		if targets[m.Target] {
			return fmt.Errorf("%s: duplicate mount target %s", path, m.Target)
		}
		targets[m.Target] = true
	}
	return nil
}
