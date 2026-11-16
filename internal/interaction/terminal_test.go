package interaction

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReadLine(t *testing.T) {
	var out strings.Builder
	in := strings.NewReader("  yes \nvm\n")
	terminal := Terminal{In: in, Out: &out, IsTTY: func() bool { return true }}
	for _, want := range []string{"yes", "vm"} {
		answer, err := terminal.ReadLine(context.Background(), "prompt ")
		if err != nil || answer != want {
			t.Fatal(answer, err)
		}
	}
	if out.String() != "prompt prompt " {
		t.Fatal(out.String())
	}
	if _, err := terminal.ReadLine(context.Background(), ""); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
}
func TestReadLineRequiresTerminalAndHonorsCancellation(t *testing.T) {
	terminal := Terminal{In: strings.NewReader("yes\n"), Out: io.Discard, IsTTY: func() bool { return false }}
	if _, err := terminal.ReadLine(context.Background(), ""); err == nil {
		t.Fatal("nonterminal approved")
	}
	terminal.IsTTY = func() bool { return true }
	in, out := io.Pipe()
	defer in.Close()
	defer out.Close()
	terminal.In = in
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := terminal.ReadLine(ctx, ""); done <- err }()
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
