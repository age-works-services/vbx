package commands

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"
	"vbx/internal/command"
)

func init() { register(newListVMsCommand) }
func newListVMsCommand(d Dependencies) command.Command {
	inventory, out := d.Inventory, d.Out
	return command.Handler{Definition: command.Spec{Name: "list-vms", Summary: "List known VBX VMs, project paths, and status", Usage: "list-vms"}, Execute: func(ctx context.Context, in command.Invocation) error {
		instances, err := inventory.List(ctx)
		if err != nil {
			return err
		}
		sort.Slice(instances, func(i, j int) bool { return instances[i].Name < instances[j].Name })
		table := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(table, "NAME\tPROJECT\tSTATUS")
		for _, instance := range instances {
			if !strings.HasPrefix(instance.Name, "vbx-") {
				continue
			}
			path := "-"
			// VBX renders the primary project mount before any additional mounts.
			for _, mount := range instance.Mounts {
				if strings.HasPrefix(mount.MountPoint, "/workspaces/") {
					path = mount.Location
					break
				}
			}
			if strings.ContainsFunc(path, unicode.IsControl) {
				path = strconv.Quote(path)
			}
			fmt.Fprintf(table, "%s\t%s\t%s\n", instance.Name, path, instance.Status)
		}
		return table.Flush()
	}}
}
