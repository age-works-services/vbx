package host

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

type Checker struct{}

func (Checker) Check(ctx context.Context) error {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("VBX supports Linux x86_64 hosts; see docs/manuals/host-setup.md in the project repository")
	}
	if os.Geteuid() == 0 {
		return fmt.Errorf("run as your normal host user, without sudo")
	}
	f, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("the host user needs read/write access to /dev/kvm: %w", err)
	}
	f.Close()
	for _, name := range []string{"limactl", "qemu-system-x86_64", "qemu-img", "ssh"} {
		if _, err := exec.LookPath(name); err != nil {
			return fmt.Errorf("missing host tool %s: %w", name, err)
		}
	}
	return nil
}
