package provision

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"vbx/internal/module"
	"vbx/internal/state"
)

func Quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

var exportPrefix = regexp.MustCompile(`^export\s+`)

func Environment(files []string) (string, error) {
	values := map[string]string{}
	for _, path := range files {
		f, e := os.Open(path)
		if e != nil {
			return "", fmt.Errorf("cannot read environment file %s", path)
		}
		scan := bufio.NewScanner(f)
		scan.Buffer(make([]byte, 4096), 1024*1024)
		line := 0
		for scan.Scan() {
			line++
			s := strings.TrimSpace(scan.Text())
			if s == "" || strings.HasPrefix(s, "#") {
				continue
			}
			s = exportPrefix.ReplaceAllString(s, "")
			key, v, ok := strings.Cut(s, "=")
			invalid := func() (string, error) {
				f.Close()
				return "", fmt.Errorf("%s:%d: invalid or reserved environment assignment", path, line)
			}
			if !ok || !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(key) || regexp.MustCompile(`^(HOME|USER|LOGNAME|SHELL|PATH|BASH_ENV|ENV|VBX_.*|XDG_.*)$`).MatchString(key) {
				return invalid()
			}
			v = strings.TrimSpace(v)
			if strings.HasPrefix(v, "'") || strings.HasPrefix(v, "\"") {
				q := v[0]
				end := strings.IndexByte(v[1:], q)
				if end < 0 {
					return invalid()
				}
				end++
				tail := strings.TrimSpace(v[end+1:])
				if tail != "" && !strings.HasPrefix(tail, "#") {
					return invalid()
				}
				v = v[1:end]
				if q == '"' && strings.Contains(v, "\\") {
					return invalid()
				}
			} else {
				for i := 1; i < len(v); i++ {
					if v[i] == '#' && (v[i-1] == ' ' || v[i-1] == '\t') {
						v = strings.TrimSpace(v[:i])
						break
					}
				}
			}
			for _, c := range v {
				if c < 32 || c == 127 {
					return invalid()
				}
			}
			values[key] = v
		}
		e = scan.Err()
		f.Close()
		if e != nil {
			return "", fmt.Errorf("cannot read environment file %s", path)
		}
	}
	keys := []string{}
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&out, "export %s=%s\n", k, Quote(values[k]))
	}
	return out.String(), nil
}

// Build captures all mutable inputs once. The caller publishes the archive atomically.
func Build(w io.Writer, assets fs.FS, record state.Record, local, project string) error {
	c := record.Config
	_, err := module.Load(assets, c.Modules)
	if err != nil {
		return err
	}
	if err = record.Validate(); err != nil {
		return err
	}
	bindings, mappings := record.GuestRecords()
	tw := tar.NewWriter(w)
	defer tw.Close()
	add := func(name string, data []byte, mode int64) error {
		if e := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(data)), Typeflag: tar.TypeReg}); e != nil {
			return e
		}
		_, e := tw.Write(data)
		return e
	}
	for _, pair := range [][2]string{{"provision-guest.sh", "provision.sh"}, {"apt-install.sh", "apt-install.sh"}, {"guest-shell-system.sh", "guest-shell-system.sh"}, {"npm-policy.sh", "npm-policy.sh"}, {"persistence.sh", "persistence.sh"}} {
		b, e := fs.ReadFile(assets, "scripts/"+pair[0])
		if e != nil {
			return e
		}
		if e = add(pair[1], b, 0600); e != nil {
			return e
		}
	}
	for _, module := range c.Modules {
		root := "modules/" + module

		e := fs.WalkDir(assets, root, func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return tw.WriteHeader(&tar.Header{Name: path + "/", Mode: 0755, Typeflag: tar.TypeDir})
			}
			b, e := fs.ReadFile(assets, path)
			if e != nil {
				return e
			}
			return add(path, b, 0644)
		})
		if e != nil {
			return e
		}
	}
	if err = WalkHome(local, project, c.Persistence, func(entry HomeEntry) error {
		if entry.Info == nil {
			return tw.WriteHeader(&tar.Header{Name: entry.ArchiveName() + "/", Mode: 0700, Typeflag: tar.TypeDir})
		}
		h, err := tar.FileInfoHeader(entry.Info, entry.Link)
		if err != nil {
			return err
		}
		h.Name = entry.ArchiveName()
		h.Uid = 0
		h.Gid = 0
		h.Uname = ""
		h.Gname = ""
		if err = tw.WriteHeader(h); err != nil {
			return err
		}
		if entry.Info.Mode().IsRegular() {
			f, err := os.Open(entry.Path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(tw, f)
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		}
		return nil
	}); err != nil {
		return err
	}
	setup := filepath.Join(project, ".vbx/setup.sh")
	if _, e := os.Lstat(setup); e == nil {
		if st, e := os.Stat(setup); e != nil || !st.Mode().IsRegular() {
			return fmt.Errorf("project setup must be a regular file")
		}
		b, e := os.ReadFile(setup)
		if e != nil {
			return e
		}
		if e = add("project-setup.sh", b, 0600); e != nil {
			return e
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	env, e := Environment(c.EnvFiles)
	if e != nil {
		return e
	}
	for _, v := range []struct{ name, text string }{{"bindings.tsv", bindings}, {"mappings.tsv", mappings}, {"environment.sh", env}, {"modules.list", lines(c.Modules)}, {"apt.txt", lines(c.APT)}} {
		if e = add(v.name, []byte(v.text), 0600); e != nil {
			return e
		}
	}
	return tw.Close()
}
