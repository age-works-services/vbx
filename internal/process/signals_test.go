package process

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestLifecycleSignals(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			cause := SignalError{Signal: sig}
			if cause.ExitCode() != 128+int(sig) {
				t.Fatal(cause.ExitCode())
			}
			d := t.TempDir()
			ready, received := filepath.Join(d, "ready"), filepath.Join(d, "received")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			done := make(chan error, 1)
			go func() {
				done <- (OS{}).Run(ctx, nil, io.Discard, io.Discard, "bash", "-c", `trap 'echo HUP > "$2"; exit 0' HUP
trap 'echo INT > "$2"; exit 0' INT
trap 'echo TERM > "$2"; exit 0' TERM
: > "$1"
while :; do sleep 0.03; done`, "fixture", ready, received)
			}()
			deadline := time.After(3 * time.Second)
			for {
				if _, err := os.Stat(ready); err == nil {
					break
				}
				select {
				case <-deadline:
					t.Fatal("child not ready")
				case <-time.After(5 * time.Millisecond):
				}
			}
			cancel(cause)
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("child not canceled")
			}
			b, err := os.ReadFile(received)
			want := map[syscall.Signal]string{syscall.SIGHUP: "HUP\n", syscall.SIGINT: "INT\n", syscall.SIGTERM: "TERM\n"}[sig]
			if err != nil || string(b) != want {
				t.Fatalf("signal %s: got %q, %v", strconv.Itoa(int(sig)), b, err)
			}
		})
	}
}
