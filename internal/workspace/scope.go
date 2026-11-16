package workspace

import (
	"context"
	"io"
	"io/fs"
	"os"
	"sync"
	"vbx/internal/backend/lima"
	"vbx/internal/project"
	"vbx/internal/state"
)

type backend interface {
	List(context.Context) ([]lima.Instance, error)
	Exists(context.Context, string) (bool, error)
	Start(context.Context, string, string) error
	Validate(context.Context, string) error
	Shell(context.Context, string, string, io.Reader, ...string) error
	SSHConfigFile(context.Context, string) (string, error)
}
type hostChecks interface{ Check(context.Context) error }
type lineReader interface {
	ReadLine(context.Context, string) (string, error)
}
type Service struct {
	Backend     backend
	Host        hostChecks
	Interaction lineReader
	Assets      fs.FS
	Paths       state.Paths
}
type Scope struct {
	Project               project.Project
	InstanceDir, SSHLink  string
	VMExists, StateExists bool
	release               func()
	once                  sync.Once
}

func (s *Scope) Release() {
	s.once.Do(func() {
		if s.release != nil {
			s.release()
		}
	})
}
func (s *Service) Acquire(ctx context.Context, path string) (*Scope, error) {
	if ctx.Err() != nil {
		return nil, context.Cause(ctx)
	}
	p, err := project.Resolve(path)
	if err != nil {
		return nil, err
	}
	paths, err := s.Paths.Instance(p.Name)
	if err != nil {
		return nil, err
	}
	base, err := s.Paths.StateDir()
	if err != nil {
		return nil, err
	}
	unlock, err := state.Lock(base, p.Name)
	if err != nil {
		return nil, err
	}
	scope := &Scope{Project: p, InstanceDir: paths.Dir, SSHLink: paths.SSHLink, release: unlock}
	exists, err := s.Backend.Exists(ctx, p.Name)
	if err != nil {
		scope.Release()
		return nil, err
	}
	scope.VMExists = exists
	if _, err = os.Lstat(paths.Dir); err == nil {
		scope.StateExists = true
	} else if !os.IsNotExist(err) {
		scope.Release()
		return nil, err
	}
	return scope, nil
}
