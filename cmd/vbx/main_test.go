package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"vbx/internal/state"
)

func TestHangup(t *testing.T) {
	if os.Getenv("VBX_HUP_HELPER") == "1" {
		os.Args = []string{"vbx", "--project", os.Getenv("VBX_HUP_PROJECT"), "stop"}
		os.Exit(run())
	}
	d := t.TempDir()
	bin := filepath.Join(d, "bin")
	os.Mkdir(bin, 0700)
	script := `#!/bin/bash
trap 'echo HUP > "$VBX_HUP_PROJECT/received"; exit 0' HUP
: > "$VBX_HUP_PROJECT/ready"
while :; do sleep 0.03; done
`
	if err := os.WriteFile(filepath.Join(bin, "limactl"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestHangup$")
	cmd.Env = append(os.Environ(), "VBX_HUP_HELPER=1", "VBX_HUP_PROJECT="+d, "PATH="+bin+":"+os.Getenv("PATH"), "XDG_STATE_HOME="+filepath.Join(d, "state"), "XDG_CONFIG_HOME="+filepath.Join(d, "config"))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	defer func() { _ = cmd.Process.Kill() }()
	deadline := time.After(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(d, "ready")); err == nil {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("helper exited early: %v", err)
		case <-deadline:
			t.Fatal("helper not ready")
		case <-time.After(10 * time.Millisecond):
		}
	}
	if err := cmd.Process.Signal(syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 129 {
			t.Fatalf("got %v, expected exit 129", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("hangup did not exit")
	}
	received, err := os.ReadFile(filepath.Join(d, "received"))
	if err != nil || strings.TrimSpace(string(received)) != "HUP" {
		t.Fatal(string(received), err)
	}
	locks, err := filepath.Glob(filepath.Join(d, "state/vbx/locks/*.lock"))
	if err != nil || len(locks) != 1 {
		t.Fatal(locks, err)
	}
	name := strings.TrimSuffix(filepath.Base(locks[0]), ".lock")
	unlock, err := state.Lock(filepath.Join(d, "state/vbx"), name)
	if err != nil {
		t.Fatal("lock retained after hangup", err)
	}
	unlock()
}
