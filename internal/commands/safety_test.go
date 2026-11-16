package commands

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"vbx/internal/backend/lima"
	"vbx/internal/command"
	"vbx/internal/config"
	"vbx/internal/process"
	"vbx/internal/state"
)

func TestRenderAndSSH(t *testing.T) {
	a, _, p, _ := fixture(t)
	c := config.Defaults()
	c.CPUs = "5"
	c.Memory = "8GiB"
	c.Disk = "40GiB"
	p.Path = "/tmp/a 'quoted' path"
	c.Mounts = []config.Mount{{Source: "/tmp/api's", Target: "/workspaces/api", Readonly: true}}
	out, err := lima.Render(a.Assets, c, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"cpus: 5\n", "memory: 8GiB\n", "disk: 40GiB\n", "location: '/tmp/a ''quoted'' path'", "location: '/tmp/api''s'", "mountPoint: '/workspaces/api'", "writable: false"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
	var b bytes.Buffer
	a.Out = &b
	if err = newShowHostSSHConfigLineCommand(a.dependencies()).Run(context.Background(), command.Invocation{}); err != nil {
		t.Fatal(err)
	}
	local, _ := localDir()
	want := fmt.Sprintf("Include \"%s/ssh/*.conf\"\n", local)
	if b.String() != want {
		t.Fatal(b.String(), want)
	}
}

func TestRemovalSafety(t *testing.T) {
	for _, tc := range []struct {
		name, status, fail string
		tty, ok            bool
		want               []string
	}{{"running", "Running", "", true, true, []string{"stop", "delete"}}, {"failed stop", "Running", "stop", true, false, []string{"stop"}}, {"broken", "Broken", "", true, false, nil}, {"not a tty", "Stopped", "", false, false, nil}} {
		t.Run(tc.name, func(t *testing.T) {
			a, f, p, dir := fixture(t)
			if err := runStart(context.Background(), a, p.Path, "workspace-only"); err != nil {
				t.Fatal(err)
			}
			f.calls = nil
			f.status = tc.status
			f.fail = tc.fail
			a.IsTTY = func() bool { return tc.tty }
			a.In = strings.NewReader(p.Name + "\n")
			err := runRemove(context.Background(), a, p.Path)
			if (err == nil) != tc.ok {
				t.Fatal(err)
			}
			var ops []string
			for _, s := range f.calls {
				if strings.HasPrefix(s, "stop ") {
					ops = append(ops, "stop")
				}
				if strings.HasPrefix(s, "delete ") {
					ops = append(ops, "delete")
				}
			}
			if !reflect.DeepEqual(ops, tc.want) {
				t.Fatal(ops, tc.want)
			}
			if !tc.ok {
				if _, err = os.Stat(filepath.Join(dir, "config")); err != nil {
					t.Fatal("state lost", err)
				}
			}
		})
	}
}

func TestValidationFailureBlocksCreation(t *testing.T) {
	a, f, p, dir := fixture(t)
	f.fail = "validate"
	if err := runStart(context.Background(), a, p.Path, "workspace-only"); err == nil {
		t.Fatal("validation failure ignored")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("state created", err)
	}
	for _, s := range f.calls {
		if strings.HasPrefix(s, "start ") {
			t.Fatal("started VM")
		}
	}
}

func TestBrokenSnapshots(t *testing.T) {
	for _, kind := range []string{"missing snapshot", "corrupt snapshot", "unknown status", "missing status", "incomplete without archive"} {
		t.Run(kind, func(t *testing.T) {
			a, f, p, dir := fixture(t)
			os.MkdirAll(dir, 0700)
			f.exists = p.Name
			c := config.Defaults()
			c.Folder = p.Primary
			b, _ := state.Encode(c)
			if kind != "missing snapshot" {
				if kind == "corrupt snapshot" {
					b = []byte("broken")
				}
				os.WriteFile(filepath.Join(dir, "config"), b, 0600)
			}
			if kind != "missing status" {
				status := "ready\n"
				if kind == "unknown status" {
					status = "unknown\n"
				}
				if kind == "incomplete without archive" {
					status = "setup-failed\n"
				}
				os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0600)
			}
			if err := runStart(context.Background(), a, p.Path, "workspace-only"); err == nil {
				t.Fatal("broken state accepted")
			}
			for _, s := range f.calls {
				if strings.HasPrefix(s, "start ") || strings.HasPrefix(s, "delete ") {
					t.Fatal("mutated VM", s)
				}
			}
		})
	}
}

func TestSecretFreeFailure(t *testing.T) {
	a, f, p, _ := fixture(t)
	local, _ := localDir()
	os.MkdirAll(local, 0700)
	os.WriteFile(filepath.Join(local, "config.yaml"), []byte("version: 1\nenvFiles:\n  - .env\n"), 0600)
	const secret = "fixture-private-value"
	os.WriteFile(filepath.Join(local, ".env"), []byte("TOKEN="+secret+"\n"), 0600)
	f.fail = "shell"
	var log bytes.Buffer
	a.Out = &log
	a.Err = &log
	err := runStart(context.Background(), a, p.Path, "workspace-only")
	if err == nil {
		t.Fatal("setup succeeded")
	}
	if strings.Contains(log.String()+err.Error()+strings.Join(f.calls, "\n"), secret) {
		t.Fatal("secret leaked")
	}
}

func TestLaunchArguments(t *testing.T) {
	a, f, p, dir := fixture(t)
	if err := runStart(context.Background(), a, p.Path, "workspace-only"); err != nil {
		t.Fatal(err)
	}
	c := config.Defaults()
	c.Folder = "/workspaces"
	b, _ := state.Encode(c)
	os.WriteFile(filepath.Join(dir, "config"), b, 0600)
	for _, target := range []string{"terminal", "vscode", ""} {
		if err := runStart(context.Background(), a, p.Path, target); err != nil {
			t.Fatal(err)
		}
		want := []string{"code", "--remote", "ssh-remote+lima-" + p.Name, "/workspaces"}
		if target == "terminal" {
			want = []string{"limactl", "shell", "--workdir=" + p.Primary, p.Name, "--", "bash", "-eu", "-c", lima.GuestScript, "vbx-exec", "bash", "-l"}
		}
		if !reflect.DeepEqual(f.replacement, want) {
			t.Fatal(f.replacement, want)
		}
	}
}

type cancelAssets struct {
	fs.FS
	cancel func()
}

func (c cancelAssets) Open(name string) (fs.File, error) {
	if name == "scripts/provision-guest.sh" {
		c.cancel()
		return nil, context.Canceled
	}
	return c.FS.Open(name)
}
func TestCancellationCleansTemporaryArchive(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			a, f, p, _ := fixture(t)
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			a.Assets = cancelAssets{FS: a.Assets, cancel: func() { cancel(process.SignalError{Signal: sig}) }}
			f.hook = func(ctx context.Context, args []string) error {
				if ctx.Err() != nil {
					return context.Cause(ctx)
				}
				return nil
			}
			if err := runStart(ctx, a, p.Path, "workspace-only"); err == nil {
				t.Fatal("cancellation ignored")
			}
			entries, err := os.ReadDir(tmp)
			if err != nil || len(entries) != 0 {
				t.Fatal("temporary archive retained", entries, err)
			}
			base, _ := state.Base("XDG_STATE_HOME", ".local/state")
			unlock, err := state.Lock(base, p.Name)
			if err != nil {
				t.Fatal("lock retained", err)
			}
			unlock()
		})
	}
}
