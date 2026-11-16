package workspace

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"vbx/internal/backend/lima"
	"vbx/internal/state"
)

func (s *Service) record(scope *Scope) (state.Record, error) {
	data, err := os.ReadFile(filepath.Join(scope.InstanceDir, "config"))
	if err != nil {
		return state.Record{}, err
	}
	return state.DecodeRecord(data)
}
func (s *Service) validateData(ctx context.Context, scope *Scope, r state.Record) error {
	if err := state.ValidateData(scope.Project.Name, r); err != nil {
		return err
	}
	if len(r.Bindings) == 0 {
		return nil
	}
	instances, err := s.Backend.List(ctx)
	if err != nil {
		return err
	}
	expected := []lima.Mount{{Location: scope.Project.Path, MountPoint: scope.Project.Primary, Writable: true}}
	for _, m := range r.Config.Mounts {
		expected = append(expected, lima.Mount{Location: m.Source, MountPoint: m.Target, Writable: !m.Readonly})
	}
	for _, b := range r.Bindings {
		expected = append(expected, lima.Mount{Location: b.Host, MountPoint: b.Guest, Writable: b.Writable})
	}
	for _, instance := range instances {
		if instance.Name == scope.Project.Name {
			if instance.MountType != "9p" {
				return fmt.Errorf("VM mount type differs from captured 9p configuration: %s", scope.Project.Name)
			}
			if len(instance.Mounts) != len(expected) {
				break
			}
			for i, m := range expected {
				if instance.Mounts[i] != m {
					return fmt.Errorf("VM mounts differ from captured configuration; inspect %s", scope.Project.Name)
				}
			}
			return nil
		}
	}
	return fmt.Errorf("VM mount configuration unavailable or inconsistent: %s", scope.Project.Name)
}
func (s *Service) verifyData(ctx context.Context, scope *Scope, r state.Record, mode string) error {
	if len(r.Bindings) == 0 {
		return nil
	}
	script, err := fs.ReadFile(s.Assets, "scripts/persistence.sh")
	if err != nil {
		return err
	}
	bindings, mappings := r.GuestRecords()
	var input bytes.Buffer
	tw := tar.NewWriter(&input)
	for _, file := range []struct {
		name string
		data []byte
	}{{"persistence.sh", script}, {"bindings.tsv", []byte(bindings)}, {"mappings.tsv", []byte(mappings)}} {
		if err = tw.WriteHeader(&tar.Header{Name: file.name, Mode: 0600, Size: int64(len(file.data))}); err != nil {
			return err
		}
		if _, err = tw.Write(file.data); err != nil {
			return err
		}
	}
	if err = tw.Close(); err != nil {
		return err
	}
	return s.Backend.Shell(ctx, scope.Project.Name, "/", &input, "bash", "-eu", "-c", verifyDataScript, "vbx-persistence", mode, scope.Project.Name)
}

const verifyDataScript = `bundle=$(mktemp -d)
trap 'chmod -R u+rwX -- "$bundle"; rm -rf -- "$bundle"' EXIT
tar -xf - -C "$bundle"
bash "$bundle/persistence.sh" "$1" "$bundle" "$2"`
