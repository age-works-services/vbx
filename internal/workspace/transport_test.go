package workspace

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"vbx"
	"vbx/internal/config"
	"vbx/internal/process"
	"vbx/internal/provision"
	"vbx/internal/state"
)

func TestActualGuestTransport(t *testing.T) {
	d := t.TempDir()
	home := filepath.Join(d, "home")
	bin := filepath.Join(d, "bin")
	p := filepath.Join(d, "project")
	for _, dir := range []string{home, bin, filepath.Join(p, ".vbx")} {
		os.MkdirAll(dir, 0700)
	}
	t.Setenv("HOME", home)
	// Only expose the utilities this transport fixture needs, so optional host
	// tools cannot change its behavior.
	for _, name := range []string{"bash", "tar", "mktemp", "chmod", "rm", "mkdir", "dirname", "touch", "find", "cp", "install"} {
		path, err := exec.LookPath(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(path, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	os.WriteFile(filepath.Join(bin, "sudo"), []byte(`#!/bin/sh
set -eu
[ "$1" = -n ]; shift
case "$2" in
  */apt-install.sh) exec "$@" ;;
  */guest-shell-system.sh) exit 0 ;;
  *) exit 99 ;;
esac
`), 0700)
	// Empty APT input must exit before any system command or cleanup runs.
	os.WriteFile(filepath.Join(bin, "apt-get"), []byte("#!/bin/sh\necho unexpected-apt >&2\nexit 94\n"), 0700)
	env := filepath.Join(d, "env")
	os.WriteFile(env, []byte("ORDER=project\n"), 0600)
	// Guest project setup receives no positional arguments; record its actual cwd and bundle path.
	setup := `set -eu
[[ "$ORDER" == project ]]
printf '%s\n' "$PWD" > "$HOME/project-cwd"
printf '%s\n' "$(dirname -- "$0")" > "$HOME/bundle-path"
`
	os.WriteFile(filepath.Join(p, ".vbx/setup.sh"), []byte(setup), 0600)
	c := config.Defaults()
	c.EnvFiles = []string{env}
	var b bytes.Buffer
	if e := provision.Build(&b, vbx.Assets, state.Record{Config: c}, filepath.Join(d, "local"), p); e != nil {
		t.Fatal(e)
	}
	var errOut bytes.Buffer
	if e := (process.OS{}).Run(context.Background(), &b, io.Discard, &errOut, "bash", "-eu", "-c", "cd -- \"$1\"\n"+ProvisionScript, "vbx-provision", p, "transport"); e != nil {
		t.Fatal(e, errOut.String())
	}
	cwd, _ := os.ReadFile(filepath.Join(home, "project-cwd"))
	if strings.TrimSpace(string(cwd)) != p {
		t.Fatal(string(cwd))
	}
	staging, _ := os.ReadFile(filepath.Join(home, "bundle-path"))
	if _, e := os.Stat(strings.TrimSpace(string(staging))); !os.IsNotExist(e) {
		t.Fatal("guest staging retained")
	}
	if _, e := os.Stat(filepath.Join(home, ".local/state/vbx/provision/transport/complete")); e != nil {
		t.Fatal(e)
	}
}
