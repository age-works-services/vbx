package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"vbx/internal/config"
	"vbx/internal/project"
	"vbx/internal/state"
)

type Ready struct {
	Project project.Project
	Config  config.Config
}

func (s *Service) EnsureReady(ctx context.Context, path string) (Ready, error) {
	if err := s.Host.Check(ctx); err != nil {
		return Ready{}, err
	}
	scope, err := s.Acquire(ctx, path)
	if err != nil {
		return Ready{}, err
	}
	defer scope.Release()
	var c config.Config
	if scope.VMExists {
		c, err = s.existing(ctx, scope)
	} else if scope.StateExists {
		return Ready{}, fmt.Errorf("VM is missing but saved state exists; inspect it or run vbx remove explicitly")
	} else {
		c, err = s.create(ctx, scope)
	}
	if err != nil {
		return Ready{}, err
	}
	record, err := s.record(scope)
	if err != nil {
		return Ready{}, err
	}
	if !scope.VMExists {
		if err = s.validateData(ctx, scope, record); err != nil {
			return Ready{}, err
		}
	}
	if err = s.verifyData(ctx, scope, record, "mounts"); err != nil {
		return Ready{}, err
	}
	if err = s.provision(ctx, scope); err != nil {
		return Ready{}, err
	}
	if err = s.verifyData(ctx, scope, record, "verify"); err != nil {
		return Ready{}, err
	}
	if err = s.registerSSH(ctx, scope); err != nil {
		return Ready{}, err
	}
	if ctx.Err() != nil {
		return Ready{}, context.Cause(ctx)
	}
	return Ready{Project: scope.Project, Config: c}, nil
}
func (s *Service) existing(ctx context.Context, scope *Scope) (c config.Config, err error) {
	p, dir := scope.Project, scope.InstanceDir
	b, e := os.ReadFile(filepath.Join(dir, "config"))
	if e != nil {
		return c, fmt.Errorf("VM snapshot unavailable; inspect with limactl shell --workdir=/ %s and preserve guest-local data before removal: %w", p.Name, e)
	}
	record, e := state.DecodeRecord(b)
	c = record.Config
	if e != nil {
		return c, fmt.Errorf("VM snapshot invalid; inspect with limactl shell --workdir=/ %s and preserve guest-local data before removal: %w", p.Name, e)
	}
	status, err := os.ReadFile(filepath.Join(dir, "status"))
	if err != nil {
		return c, fmt.Errorf("VM status unavailable; inspect saved state: %w", err)
	}
	switch strings.TrimSpace(string(status)) {
	case "ready":
	case "creating", "setup", "setup-failed":
		if st, err := os.Stat(filepath.Join(dir, "provision.tar")); err != nil || !st.Mode().IsRegular() {
			return c, fmt.Errorf("incomplete VM has no usable captured setup archive; inspect or explicitly remove it")
		}
	default:
		return c, fmt.Errorf("unknown VM setup status; inspect saved state")
	}
	if e = s.validateData(ctx, scope, record); e != nil {
		return c, e
	}
	if e = s.Backend.Start(ctx, p.Name, ""); e != nil {
		return c, e
	}
	return c, nil
}
