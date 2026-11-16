package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vbx/internal/cli"
)

type setupOutputFailure struct{ err error }

func (w setupOutputFailure) Write([]byte) (int, error) { return 0, w.err }

func setupInspectionFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	project := filepath.Join(root, "project")
	local := filepath.Join(root, "config/vbx")
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	for _, dir := range []string{local, filepath.Join(project, ".vbx")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return root, local, project
}
func setupFile(t *testing.T, file, data string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(data), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, mode); err != nil {
		t.Fatal(err)
	}
}
func TestShowProjectEffectiveSetup(t *testing.T) {
	root, local, project := setupInspectionFixture(t)
	setupFile(t, filepath.Join(local, "config.yaml"), "version: 1\nmodules:\n  - pi-project-persistent-sessions\nenvFiles:\n  - .env\n", 0600)
	setupFile(t, filepath.Join(local, ".env"), "SECRET=environment-secret\n", 0600)
	setupFile(t, filepath.Join(project, ".vbx/config.yaml"), "version: 1\nresources:\n  cpus: 5\npersistence:\n  paths:\n    - guest: ~/.tool/sessions\n      data: sessions\n", 0600)
	setupFile(t, filepath.Join(local, "home/.gitconfig"), "global-secret", 0600)
	setupFile(t, filepath.Join(project, ".vbx/home/.gitconfig"), "project-secret", 0400)
	setupFile(t, filepath.Join(local, "home/.config/tool/global.json"), "private-global-json", 0600)
	setupFile(t, filepath.Join(project, ".vbx/home/.config/tool/project.json"), "private-project-json", 0640)
	// Metadata inspection must not need permission to read home file contents.
	setupFile(t, filepath.Join(project, ".vbx/home/unreadable"), "unreadable-secret", 0000)
	link := filepath.Join(project, ".vbx/home/config-link")
	if err := os.Symlink(".config/tool", link); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	var baseline string
	for _, args := range [][]string{{"show-project-effective-setup"}, {"show-project-effective-setup", "--project", project}, {"--project", project, "show-project-effective-setup"}} {
		var out strings.Builder
		if err := cli.Run(context.Background(), args, New(Dependencies{Out: &out}), &out); err != nil {
			t.Fatal(err)
		}
		got := out.String()
		for _, want := range []string{"Configuration\n", "Home overlay and persistence\n", "cpus: 5", `"/home/dev/.gitconfig" [file -r--------]`, `source: project ".vbx/home/.gitconfig"`, `overrides: global "home/.gitconfig"`, `merges: global "home/.config/tool"`, `"/home/dev/.config/tool/global.json"`, `"/home/dev/.config/tool/project.json"`, `"/home/dev/config-link" -> ".config/tool"`, `"/home/dev/.pi/agent/sessions" -> "/mnt/vbx-data/pi-project-persistent-sessions/sessions"`, `"/home/dev/.tool/sessions" -> "/mnt/vbx-config-data/sessions"`, "values hidden", "Installers and project setup"} {
			if !strings.Contains(got, want) {
				t.Fatalf("missing %q in:\n%s", want, got)
			}
		}
		for _, secret := range []string{"global-secret", "project-secret", "private-global-json", "private-project-json", "unreadable-secret", "environment-secret"} {
			if strings.Contains(got, secret) {
				t.Fatal("file contents leaked", secret)
			}
		}
		if baseline != "" && baseline != got {
			t.Fatal("nondeterministic output")
		}
		baseline = got
	}
	for _, dir := range []string{"data", "state"} {
		if _, err := os.Stat(filepath.Join(root, dir)); !os.IsNotExist(err) {
			t.Fatal("inspection changed lifecycle or retained state", dir, err)
		}
	}
}
func TestShowProjectEffectiveSetupEmpty(t *testing.T) {
	_, _, project := setupInspectionFixture(t)
	var out strings.Builder
	if err := cli.Run(context.Background(), []string{"show-project-effective-setup", "--project", project}, New(Dependencies{Out: &out}), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No home overlay entries.") || !strings.Contains(out.String(), "/home/dev/.config/vbx/environment.sh") {
		t.Fatal(out.String())
	}
}
func TestShowProjectEffectiveSetupErrors(t *testing.T) {
	for _, scenario := range []string{"configuration", "persistence", "managed environment", "state", "non-directory ancestor", "file-directory"} {
		t.Run(scenario, func(t *testing.T) {
			_, local, project := setupInspectionFixture(t)
			switch scenario {
			case "configuration":
				setupFile(t, filepath.Join(project, ".vbx/config.yaml"), "version: 1\nresources:\n  cpus: 0\n", 0600)
			case "persistence":
				setupFile(t, filepath.Join(project, ".vbx/config.yaml"), "version: 1\npersistence:\n  paths:\n    - guest: ~/.tool\n      data: tool\n", 0600)
				setupFile(t, filepath.Join(local, "home/.tool/file"), "secret", 0600)
			case "managed environment":
				setupFile(t, filepath.Join(local, "home/.config/vbx/environment.sh"), "secret", 0600)
			case "state":
				setupFile(t, filepath.Join(local, "home/.local/state/vbx/file"), "secret", 0600)
			case "non-directory ancestor":
				setupFile(t, filepath.Join(local, "home/target"), "secret", 0600)
				if err := os.Symlink("target", filepath.Join(local, "home/link")); err != nil {
					t.Fatal(err)
				}
				setupFile(t, filepath.Join(project, ".vbx/home/link/child"), "secret", 0600)
			case "file-directory":
				setupFile(t, filepath.Join(local, "home/collision/file"), "secret", 0600)
				setupFile(t, filepath.Join(project, ".vbx/home/collision"), "secret", 0600)
			}
			var out strings.Builder
			if err := cli.Run(context.Background(), []string{"show-project-effective-setup", "--project", project}, New(Dependencies{Out: &out}), nil); err == nil || out.Len() != 0 {
				t.Fatal("invalid overlay produced a report", err, out.String())
			}
		})
	}
	var help strings.Builder
	if err := cli.Run(context.Background(), []string{"show-project-effective-setup", "--help", "--project", "/missing-project"}, New(Dependencies{}), &help); err != nil || !strings.Contains(help.String(), "show-project-effective-setup") {
		t.Fatal(err, help.String())
	}
	_, _, project := setupInspectionFixture(t)
	failure := errors.New("write failed")
	if err := cli.Run(context.Background(), []string{"show-project-effective-setup", "--project", project}, New(Dependencies{Out: setupOutputFailure{failure}}), nil); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out strings.Builder
	if err := cli.Run(ctx, []string{"show-project-effective-setup", "--project", project}, New(Dependencies{Out: &out}), nil); !errors.Is(err, context.Canceled) || out.Len() != 0 {
		t.Fatal(err, out.String())
	}
}
