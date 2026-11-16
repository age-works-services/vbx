package awscli_test

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"vbx"
	"vbx/internal/config"
	"vbx/internal/provision"
	"vbx/internal/state"
)

func TestAWSCLIModuleCapture(t *testing.T) {
	local, project := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(local, "config.yaml"), []byte("version: 1\nmodules:\n  - aws-cli\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Resolve(project, "/workspaces/project", local)
	if err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	record := state.Record{Config: c, Bindings: []state.Binding{{Module: "aws-cli", Host: "/test-data/aws-cli", Guest: "/mnt/vbx-data/aws-cli", Source: "mount1", Writable: true}}}
	if err = provision.Build(&archive, vbx.Assets, record, local, project); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	reader := tar.NewReader(&archive)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		files[header.Name] = string(data)
	}
	requirements, err := vbx.Assets.ReadFile("modules/aws-cli/apt.txt")
	if err != nil {
		t.Fatal(err)
	}
	if files["modules.list"] != "aws-cli\n" || files["apt.txt"] != string(requirements) || files["modules/aws-cli/setup.sh"] == "" || files["modules/aws-cli/aws-cli-signing-key.asc"] == "" {
		t.Fatal(files)
	}
}
