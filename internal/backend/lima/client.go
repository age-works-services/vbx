// Package lima owns the Lima process protocol and configuration format.
package lima

import (
	"bytes"
	"context"
	"io"
	"strings"
	"vbx/internal/process"
)

type Client struct {
	Runner   process.Runner
	In       io.Reader
	Out, Err io.Writer
}

func (c *Client) run(ctx context.Context, in io.Reader, out io.Writer, args ...string) error {
	return c.Runner.Run(ctx, in, out, c.Err, "limactl", args...)
}
func (c *Client) capture(ctx context.Context, args ...string) (string, error) {
	var b bytes.Buffer
	err := c.run(ctx, nil, &b, args...)
	return strings.TrimRight(b.String(), "\n"), err
}
func (c *Client) Exists(ctx context.Context, name string) (bool, error) {
	names, err := c.capture(ctx, "list", "--format", "{{.Name}}")
	if err != nil {
		return false, err
	}
	for _, n := range strings.Split(names, "\n") {
		if n == name {
			return true, nil
		}
	}
	return false, nil
}
func (c *Client) Status(ctx context.Context, name string) (string, error) {
	return c.capture(ctx, "list", "--format", "{{.Status}}", name)
}
func (c *Client) SSHConfigFile(ctx context.Context, name string) (string, error) {
	return c.capture(ctx, "list", "--format", "{{.SSHConfigFile}}", name)
}
func (c *Client) Start(ctx context.Context, name, template string) error {
	if template == "" {
		return c.run(ctx, c.In, c.Out, "start", "--tty=false", name)
	}
	return c.run(ctx, c.In, c.Out, "start", "--tty=false", "--name="+name, template)
}
func (c *Client) Stop(ctx context.Context, name string) error {
	return c.run(ctx, c.In, c.Out, "stop", name)
}
func (c *Client) Delete(ctx context.Context, name string) error {
	return c.run(ctx, c.In, c.Out, "delete", "--tty=false", name)
}
func (c *Client) Validate(ctx context.Context, path string) error {
	return c.run(ctx, nil, c.Out, "validate", path)
}
func (c *Client) Shell(ctx context.Context, name, workdir string, in io.Reader, args ...string) error {
	return c.run(ctx, in, c.Out, append([]string{"shell", "--workdir=" + workdir, name, "--"}, args...)...)
}
