package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func (s *Service) registerSSH(ctx context.Context, scope *Scope) error {
	ssh := scope.SSHLink
	sshSource, e := s.Backend.SSHConfigFile(ctx, scope.Project.Name)
	if e != nil {
		return e
	}
	if st, e := os.Stat(sshSource); e != nil || !st.Mode().IsRegular() {
		return fmt.Errorf("Lima did not report a usable SSH configuration file")
	}
	if e = os.MkdirAll(filepath.Dir(ssh), 0700); e != nil {
		return e
	}
	tmp, e := os.CreateTemp(filepath.Dir(ssh), ".ssh-*")
	if e != nil {
		return e
	}
	tmp.Close()
	os.Remove(tmp.Name())
	defer os.Remove(tmp.Name())
	if e = os.Symlink(sshSource, tmp.Name()); e != nil {
		return e
	}
	if e = os.Rename(tmp.Name(), ssh); e != nil {
		return e
	}
	os.Remove(tmp.Name())

	return nil
}
