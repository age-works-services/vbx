package provision

import (
	"fmt"
	"vbx/internal/config"
	"vbx/internal/state"
)

func bundleRecord(c config.Config) state.Record {
	r := state.Record{Config: c}
	for i, name := range c.Modules {
		r.Bindings = append(r.Bindings, state.Binding{Module: name, Host: "/test-data/" + name, Guest: "/mnt/vbx-data/" + name, Source: fmt.Sprintf("mount%d", 1+len(c.Mounts)+i), Writable: true})
	}
	return r
}
