package lima

import (
	"fmt"
	"io/fs"
	"strings"
	"vbx/internal/config"
	"vbx/internal/project"
	"vbx/internal/state"
)

func Render(assets fs.FS, c config.Config, p project.Project, bindings ...state.Binding) ([]byte, error) {
	b, e := fs.ReadFile(assets, "lima/default.yaml")
	if e != nil {
		return nil, e
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	var out strings.Builder
	for _, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "cpus: "):
			line = "cpus: " + c.CPUs
		case strings.HasPrefix(line, "memory: "):
			line = "memory: " + c.Memory
		case strings.HasPrefix(line, "disk: "):
			line = "disk: " + c.Disk
		case line == "mounts: [] # VBX_PROJECT_MOUNT":
			fmt.Fprintf(&out, "mounts:\n  - location: %s\n    mountPoint: %s\n    writable: true\n", quote(p.Path), quote(p.Primary))
			for _, m := range c.Mounts {
				fmt.Fprintf(&out, "  - location: %s\n    mountPoint: %s\n    writable: %t\n", quote(m.Source), quote(m.Target), !m.Readonly)
			}
			for _, b := range bindings {
				fmt.Fprintf(&out, "  - location: %s\n    mountPoint: %s\n    writable: true\n", quote(b.Host), quote(b.Guest))
			}
			continue
		}
		out.WriteString(line + "\n")
	}
	return []byte(out.String()), nil
}
