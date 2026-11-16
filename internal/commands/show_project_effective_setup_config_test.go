package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vbx"
	"vbx/internal/cli"
	"vbx/internal/config"
	"vbx/internal/project"
	"vbx/internal/state"
)

func TestSetupConfigurationDefaults(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	p, err := project.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(p.Path)
	c := config.Defaults()
	c.Folder = p.Primary
	for _, args := range [][]string{{"show-project-effective-setup"}, {"--project", p.Path, "show-project-effective-setup"}, {"show-project-effective-setup", "--project", p.Path}} {
		var out strings.Builder
		if err := cli.Run(context.Background(), args, New(Dependencies{Out: &out, Paths: state.Paths{}}), &out); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(out.String(), "Configuration\n=============\n"+string(config.Format(c))+"\nHome overlay and persistence\n") {
			t.Fatal(out.String())
		}
	}
}
func TestSetupConfigurationMerged(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	local := filepath.Join(root, "config/vbx")
	p := filepath.Join(root, "project")
	source := filepath.Join(root, "shared files")
	for _, dir := range []string{local, filepath.Join(p, ".vbx"), source} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(local, "config.yaml"):  "version: 1\nresources:\n  cpus: 3\n  memory: 8GiB\nmodules:\n  - dev-tools\napt:\n  packages:\n    - git\nenvFiles:\n  - .env\n",
		filepath.Join(local, ".env"):         "TOKEN=private-value\n",
		filepath.Join(p, ".vbx/config.yaml"): "version: 1\nresources:\n  cpus: 5\nlaunch:\n  default: terminal\nmounts:\n  - source: ../shared files\n    target: /workspaces/shared\n    readonly: true\nmodules:\n  - dev-tools\n  - node\napt:\n  packages:\n    - git\n    - 'jq (>= 1.7)'\nenvFiles:\n  - project.env\n",
		filepath.Join(p, "project.env"):      "OTHER=another-private-value\n",
	}
	for path, data := range files {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var out strings.Builder
	if err := cli.Run(context.Background(), []string{"show-project-effective-setup", "--project", p}, New(Dependencies{Out: &out}), &out); err != nil {
		t.Fatal(err)
	}
	expected := config.Config{CPUs: "5", Memory: "8GiB", Disk: "30GiB", Default: "terminal", Folder: "/workspaces", Mounts: []config.Mount{{Source: source, Target: "/workspaces/shared", Readonly: true}}, Modules: []string{"dev-tools", "node"}, APT: []string{"git", "jq (>= 1.7)"}, EnvFiles: []string{filepath.Join(local, ".env"), filepath.Join(p, "project.env")}}
	var packages []string
	for i, name := range []string{"dev-tools", "node"} {
		settings, err := config.ModuleSettings(vbx.Assets, name)
		if err != nil {
			t.Fatal(err)
		}
		packages = append(packages, settings.APT...)
		if i == 0 {
			packages = append(packages, "git")
		}
	}
	expected.APT = config.Unique(append(packages, expected.APT...))
	if !strings.HasPrefix(out.String(), "Configuration\n=============\n"+string(config.Format(expected))+"\nHome overlay and persistence\n") {
		t.Fatal(out.String())
	}
}
