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
	writeMount := func(host, guest string, writable, looseCache bool) {
		fmt.Fprintf(&out, "  - location: %s\n    mountPoint: %s\n    writable: %t\n", quote(host), quote(guest), writable)
		if looseCache {
			// Reuse 9p inodes so guest-side file watchers can see edits.
			out.WriteString("    9p:\n      cache: loose\n")
		}
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "cpus: "):
			line = "cpus: " + c.CPUs
		case strings.HasPrefix(line, "memory: "):
			line = "memory: " + c.Memory
		case strings.HasPrefix(line, "disk: "):
			line = "disk: " + c.Disk
		case line == "mounts: [] # VBX_PROJECT_MOUNT":
			out.WriteString("mounts:\n")
			writeMount(p.Path, p.Primary, true, true)
			for _, m := range c.Mounts {
				writeMount(m.Source, m.Target, !m.Readonly, !m.Readonly)
			}
			for _, b := range bindings {
				writeMount(b.Host, b.Guest, true, false)
			}
			continue
		}
		out.WriteString(line + "\n")
	}
	return []byte(out.String()), nil
}
