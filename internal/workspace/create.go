package workspace

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"vbx/internal/backend/lima"
	"vbx/internal/config"
	"vbx/internal/module"
	"vbx/internal/provision"
	"vbx/internal/state"
)

func (s *Service) create(ctx context.Context, scope *Scope) (c config.Config, err error) {
	p, dir := scope.Project, scope.InstanceDir
	local, e := s.Paths.ConfigDir()
	if e != nil {
		return c, e
	}
	c, e = config.ResolveWithAssets(p.Path, p.Primary, local, s.Assets)
	if e != nil {
		return c, e
	}
	if _, e = os.Stat(filepath.Join(p.Path, ".vbx")); e == nil {
		if e = s.approveProject(ctx); e != nil {
			return c, e
		}
	} else if !os.IsNotExist(e) {
		return c, e
	}
	defs, e := module.Load(s.Assets, c.Modules)
	if e != nil {
		return c, e
	}
	record, e := s.Paths.AllocateData(p.Name, c, defs, p.Path)
	if e != nil {
		return c, e
	}
	yaml, e := lima.Render(s.Assets, c, p, record.Bindings...)
	if e != nil {
		return c, e
	}
	temp, e := os.MkdirTemp("", "vbx-")
	if e != nil {
		return c, e
	}
	defer os.RemoveAll(temp)
	yp := filepath.Join(temp, "lima.yaml")
	if e = os.WriteFile(yp, yaml, 0600); e != nil {
		return c, e
	}
	if e = s.Backend.Validate(ctx, yp); e != nil {
		return c, e
	}
	// Build privately before publishing instance state or starting the VM.
	archive, e := os.OpenFile(filepath.Join(temp, "provision.tar"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if e != nil {
		return c, e
	}
	e = provision.Build(archive, s.Assets, record, local, p.Path)
	ce := archive.Close()
	if e != nil {
		return c, e
	}
	if ce != nil {
		return c, ce
	}
	b, e := state.EncodeRecord(record)
	if e != nil {
		return c, e
	}
	if e = os.MkdirAll(dir, 0700); e != nil {
		return c, e
	}
	for name, data := range map[string][]byte{"config": b, "lima.yaml": yaml, "status": []byte("creating\n")} {
		if e = state.Atomic(filepath.Join(dir, name), data, 0600); e != nil {
			return c, e
		}
	}
	f, e := os.Open(filepath.Join(temp, "provision.tar"))
	if e != nil {
		return c, e
	}
	dest, e := os.OpenFile(filepath.Join(dir, "provision.tar"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		f.Close()
		return c, e
	}
	_, e = io.Copy(dest, f)
	f.Close()
	ce = dest.Close()
	if e != nil {
		return c, e
	}
	if ce != nil {
		return c, ce
	}
	os.RemoveAll(temp)
	if e = s.Backend.Start(ctx, p.Name, filepath.Join(dir, "lima.yaml")); e != nil {
		return c, e
	}
	return c, nil
}
func (s *Service) approveProject(ctx context.Context) error {
	answer, err := s.Interaction.ReadLine(ctx, "Apply project .vbx mounts, modules, home files and setup (with access to guest credentials)? [y/N] ")
	if err != nil {
		return err
	}
	if answer != "y" && answer != "Y" && answer != "yes" {
		return fmt.Errorf("project configuration declined")
	}
	return nil
}
