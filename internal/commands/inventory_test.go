package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vbx/internal/backend/lima"
	"vbx/internal/cli"
	"vbx/internal/project"
)

type inventoryStub struct {
	instances []lima.Instance
	err       error
	calls     int
}

func (s *inventoryStub) List(context.Context) ([]lima.Instance, error) {
	s.calls++
	return append([]lima.Instance(nil), s.instances...), s.err
}
func TestStatusProjectSelection(t *testing.T) {
	path := t.TempDir()
	p, err := project.Resolve(path)
	if err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err = os.Symlink(path, alias); err != nil {
		t.Fatal(err)
	}
	t.Chdir(path)
	for _, status := range []string{"Running", "Stopped", "Broken", "Missing"} {
		for _, args := range [][]string{{"status"}, {"--project", alias, "status"}, {"status", "--project", path}} {
			inventory := &inventoryStub{instances: []lima.Instance{{Name: "unrelated", Status: "Running"}}}
			if status != "Missing" {
				inventory.instances = append(inventory.instances, lima.Instance{Name: p.Name, Status: status})
			}
			var out strings.Builder
			// Only the inventory and output dependencies exist. Readiness and mutations cannot run.
			cmds := New(Dependencies{Inventory: inventory, Out: &out})
			if err = cli.Run(context.Background(), args, cmds, &out); err != nil {
				t.Fatal(err)
			}
			if out.String() != p.Name+": "+status+"\n" || inventory.calls != 1 {
				t.Fatal(args, out.String(), inventory.calls)
			}
		}
	}
}
func TestListVMs(t *testing.T) {
	inventory := &inventoryStub{instances: []lima.Instance{
		{Name: "vbx-z", Status: "Broken"},
		{Name: "unrelated", Status: "Running"},
		{Name: "vbx-a", Status: "Stopped", Mounts: []lima.Mount{{Location: "/home/dev", MountPoint: "/home/dev"}, {Location: "/deleted/project with spaces", MountPoint: "/workspaces/a"}, {Location: "/extra", MountPoint: "/workspaces/extra"}}},
	}}
	var out strings.Builder
	// Global inventory works even if the selected project path no longer exists.
	args := []string{"list-vms", "--project", "/does-not-exist"}
	if err := cli.Run(context.Background(), args, New(Dependencies{Inventory: inventory, Out: &out}), &out); err != nil {
		t.Fatal(err)
	}
	output := out.String()
	for _, part := range []string{"NAME", "PROJECT", "STATUS", "vbx-a", "/deleted/project with spaces", "Stopped", "vbx-z", "Broken"} {
		if !strings.Contains(output, part) {
			t.Fatal(part, output)
		}
	}
	if strings.Contains(output, "unrelated") || strings.Contains(output, "/extra") || strings.Index(output, "vbx-a") > strings.Index(output, "vbx-z") {
		t.Fatal(output)
	}
	if !strings.Contains(output, "-  ") {
		t.Fatal("missing mount should show a dash", output)
	}
}
func TestInventoryHelpErrorsAndEmptyList(t *testing.T) {
	failure := errors.New("backend failed")
	for _, name := range []string{"status", "list-vms"} {
		var out strings.Builder
		inventory := &inventoryStub{err: failure}
		cmds := New(Dependencies{Inventory: inventory, Out: &out})
		if err := cli.Run(context.Background(), []string{name, "--help"}, cmds, &out); err != nil || inventory.calls != 0 {
			t.Fatal(err, inventory.calls)
		}
		out.Reset()
		if err := cli.Run(context.Background(), []string{name}, cmds, &out); !errors.Is(err, failure) || out.Len() != 0 {
			t.Fatal(err, out.String())
		}
		inventory.calls = 0
		if err := cli.Run(context.Background(), []string{name, "unexpected"}, cmds, &out); err == nil || inventory.calls != 0 {
			t.Fatal(err, inventory.calls)
		}
	}
	var out strings.Builder
	if err := cli.Run(context.Background(), []string{"list-vms"}, New(Dependencies{Inventory: &inventoryStub{}, Out: &out}), &out); err != nil || len(strings.Split(strings.TrimSpace(out.String()), "\n")) != 1 {
		t.Fatal(err, out.String())
	}
}
