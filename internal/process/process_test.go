package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLiteralArguments(t *testing.T) {
	var out bytes.Buffer
	arg := "a b; $(touch /unexpected)"
	e := (OS{}).Run(context.Background(), nil, &out, io.Discard, "printf", "%s", arg)
	if e != nil || out.String() != arg {
		t.Fatal(out.String(), e)
	}
}
func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	ready := filepath.Join(t.TempDir(), "ready")
	done := make(chan error, 1)
	go func() {
		done <- (OS{}).Run(ctx, nil, io.Discard, io.Discard, "bash", "-c", `trap 'exit 0' TERM; touch "$1"; while :; do sleep 0.05; done`, "fixture", ready)
	}()
	deadline := time.After(3 * time.Second)
	for {
		if _, e := os.Stat(ready); e == nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("child not ready")
		case <-time.After(time.Millisecond * 10):
		}
	}
	cancel(fmt.Errorf("terminated"))
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("child ignored graceful cancellation")
	}
}
func TestReplace(t *testing.T) {
	if os.Getenv("VBX_TEST_REPLACE") == "1" {
		if e := (OS{}).Replace("bash", []string{"-c", "exit 37"}); e != nil {
			os.Exit(99)
		}
		os.Exit(98)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestReplace$")
	cmd.Env = append(os.Environ(), "VBX_TEST_REPLACE=1")
	e := cmd.Run()
	if e == nil || !strings.Contains(e.Error(), "37") {
		t.Fatal(e)
	}
}
