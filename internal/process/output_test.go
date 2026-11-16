package process

import (
	"bytes"
	"context"
	"testing"
)

func TestRunDoesNotEchoCommands(t *testing.T) {
	for _, tc := range []struct {
		name, script, out, stderr string
		fail                      bool
	}{
		{name: "quiet", script: "true"},
		{name: "output", script: `printf 'result\n'; printf 'warning\n' >&2`, out: "result\n", stderr: "warning\n"},
		{name: "failure", script: `printf 'failed\n' >&2; exit 37`, stderr: "failed\n", fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			err := (OS{}).Run(context.Background(), nil, &out, &stderr, "bash", "-c", tc.script, "private-argument")
			if (err != nil) != tc.fail {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.String() != tc.out || stderr.String() != tc.stderr {
				t.Fatalf("stdout=%q stderr=%q", out.String(), stderr.String())
			}
		})
	}
}
