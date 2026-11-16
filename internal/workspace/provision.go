package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"vbx/internal/provision"
	"vbx/internal/state"
)

func (s *Service) provision(ctx context.Context, scope *Scope) error {
	p, dir := scope.Project, scope.InstanceDir
	var e error
	archive := filepath.Join(dir, "provision.tar")
	if _, e = os.Stat(archive); e == nil {
		status, e := os.ReadFile(filepath.Join(dir, "status"))
		if e != nil {
			return e
		}
		if strings.TrimSpace(string(status)) != "ready" {
			if e = state.Atomic(filepath.Join(dir, "status"), []byte("setup\n"), 0600); e != nil {
				return e
			}
			f, e := os.Open(archive)
			if e != nil {
				return e
			}
			e = s.Backend.Shell(ctx, p.Name, p.Primary, f, "bash", "-eu", "-c", ProvisionScript, "vbx-provision", p.Primary, p.Name)
			f.Close()
			if e != nil {
				_ = state.Atomic(filepath.Join(dir, "status"), []byte("setup-failed\n"), 0600)
				return fmt.Errorf("guest setup failed; VM kept for inspection. Use limactl shell --workdir=/ %s; preserve guest-local data before explicitly removing the VM: %w", provision.Quote(p.Name), e)
			}
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	if e = state.Atomic(filepath.Join(dir, "status"), []byte("ready\n"), 0600); e != nil {
		return e
	}
	if e = os.Remove(archive); e != nil && !os.IsNotExist(e) {
		return e
	}

	return nil
}

const ProvisionScript = `bundle=$(mktemp -d)
trap 'chmod -R u+rwX -- "$bundle"; rm -rf -- "$bundle"' EXIT
tar -xf - -C "$bundle"
bash "$bundle/provision.sh" "$bundle" "$1" "$2"`
