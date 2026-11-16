package lima

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type listRunner struct {
	output string
	err    error
	name   string
	args   []string
}

func (r *listRunner) Run(_ context.Context, _ io.Reader, out, _ io.Writer, name string, args ...string) error {
	r.name = name
	r.args = append([]string(nil), args...)
	if r.err != nil {
		return r.err
	}
	_, err := io.WriteString(out, r.output)
	return err
}
func (*listRunner) Replace(string, []string) error { panic("unexpected replacement") }
func TestList(t *testing.T) {
	r := &listRunner{output: `{"name":"vbx-a","status":"Running","config":{"mountType":"9p","mounts":[{"location":"/host/project with spaces","mountPoint":"/workspaces/a","writable":true}]},"extra":"ignored"}
{"name":"other","status":"Stopped"}
`}
	c := Client{Runner: r, Err: io.Discard}
	got, err := c.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Instance{{Name: "vbx-a", Status: "Running", MountType: "9p", Mounts: []Mount{{Location: "/host/project with spaces", MountPoint: "/workspaces/a", Writable: true}}}, {Name: "other", Status: "Stopped"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%#v", got)
	}
	if r.name != "limactl" || !reflect.DeepEqual(r.args, []string{"list", "--json", "--all-fields"}) {
		t.Fatal(r.name, r.args)
	}
	r.output = "\n  \n"
	got, err = c.List(context.Background())
	if err != nil || len(got) != 0 {
		t.Fatal(got, err)
	}
}
func TestListFailures(t *testing.T) {
	for _, output := range []string{`broken`, `{"name":"vm"}`, `{"status":"Running"}`, `null`, `{"name":"vm","status":"Running"} garbage`} {
		c := Client{Runner: &listRunner{output: output}, Err: io.Discard}
		if _, err := c.List(context.Background()); err == nil {
			t.Fatal("accepted", output)
		}
	}
	failure := errors.New("Lima unavailable")
	c := Client{Runner: &listRunner{err: failure}, Err: io.Discard}
	if _, err := c.List(context.Background()); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Runner = &listRunner{err: ctx.Err()}
	if _, err := c.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	// Credentials and malformed response contents should stay out of decode errors.
	c.Runner = &listRunner{output: `{"name":"private-value"`}
	if _, err := c.List(context.Background()); err == nil || strings.Contains(err.Error(), "private-value") {
		t.Fatal(err)
	}
}
