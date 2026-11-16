package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Format prints resolved configuration using the supported configuration schema.
func Format(c Config) []byte {
	var out strings.Builder
	quote := func(value string) string { data, _ := json.Marshal(value); return string(data) }
	fmt.Fprintf(&out, "version: 1\nresources:\n  cpus: %s\n  memory: %s\n  disk: %s\nlaunch:\n  default: %s\n  folder: %s\n", c.CPUs, quote(c.Memory), quote(c.Disk), quote(c.Default), quote(c.Folder))
	if len(c.Mounts) == 0 {
		out.WriteString("mounts: []\n")
	} else {
		out.WriteString("mounts:\n")
		for _, mount := range c.Mounts {
			fmt.Fprintf(&out, "  - source: %s\n    target: %s\n    readonly: %t\n", quote(mount.Source), quote(mount.Target), mount.Readonly)
		}
	}
	list := func(key, indent string, values []string) {
		if len(values) == 0 {
			fmt.Fprintf(&out, "%s%s: []\n", indent, key)
			return
		}
		fmt.Fprintf(&out, "%s%s:\n", indent, key)
		for _, value := range values {
			fmt.Fprintf(&out, "%s  - %s\n", indent, quote(value))
		}
	}
	list("modules", "", c.Modules)
	out.WriteString("persistence:\n")
	if len(c.Persistence) == 0 {
		out.WriteString("  paths: []\n")
	} else {
		out.WriteString("  paths:\n")
		for _, p := range c.Persistence {
			fmt.Fprintf(&out, "    - guest: %s\n      data: %s\n", quote(p.Guest), quote(p.Data))
		}
	}
	out.WriteString("apt:\n")
	list("packages", "  ", c.APT)
	list("envFiles", "", c.EnvFiles)
	return []byte(out.String())
}
