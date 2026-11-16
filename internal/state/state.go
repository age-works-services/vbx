package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unicode/utf8"
	"vbx/internal/config"
)

func Base(variable, suffix string) (string, error) {
	p := os.Getenv(variable)
	if p == "" {
		h, e := os.UserHomeDir()
		if e != nil {
			return "", e
		}
		p = filepath.Join(h, suffix)
	}
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("%s must be absolute", variable)
	}
	return filepath.Join(p, "vbx"), nil
}
func Atomic(path string, data []byte, mode os.FileMode) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".vbx-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(mode); e == nil {
		_, e = f.Write(data)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(f.Name(), path)
}
func Lock(base, name string) (func(), error) {
	dir := filepath.Join(base, "locks")
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(filepath.Join(dir, name+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, fmt.Errorf("another operation is using %s", name)
	}
	return func() { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
}

type snapshot struct {
	Version int
	Config  config.Config
}

func Encode(c config.Config) ([]byte, error) {
	values := []string{c.CPUs, c.Memory, c.Disk, c.Default, c.Folder}
	values = append(values, c.Modules...)
	values = append(values, c.EnvFiles...)
	values = append(values, c.APT...)
	for _, m := range c.Mounts {
		values = append(values, m.Source, m.Target)
	}
	for _, v := range values {
		if !utf8.ValidString(v) {
			return nil, fmt.Errorf("snapshots require UTF-8 configuration paths and values")
		}
	}
	return json.Marshal(snapshot{1, c})
}
func Decode(b []byte) (config.Config, error) {
	var c config.Config
	var saved snapshot
	if !utf8.Valid(b) {
		return c, fmt.Errorf("invalid snapshot encoding")
	}
	if err := json.Unmarshal(b, &saved); err != nil {
		return c, err
	}
	if saved.Version != 1 {
		return c, fmt.Errorf("unknown snapshot version")
	}
	c = saved.Config
	if c.CPUs == "" || c.Memory == "" || c.Disk == "" || c.Default == "" || c.Folder == "" {
		return c, fmt.Errorf("incomplete snapshot")
	}
	if c.Default != "vscode" && c.Default != "terminal" && c.Default != "workspace-only" {
		return c, fmt.Errorf("invalid saved launch target")
	}
	return c, nil
}
