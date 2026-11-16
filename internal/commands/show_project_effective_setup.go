package commands

import (
	"context"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"vbx/internal/command"
	"vbx/internal/config"
	"vbx/internal/project"
	"vbx/internal/provision"
)

func init() { register(newShowProjectEffectiveSetupCommand) }
func newShowProjectEffectiveSetupCommand(d Dependencies) command.Command {
	paths, out := d.Paths, d.Out
	return command.Handler{Definition: command.Spec{Name: "show-project-effective-setup", Summary: "Inspect desired configuration, home overlays, and persistence", Usage: "show-project-effective-setup"}, Execute: func(ctx context.Context, in command.Invocation) error {
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		p, err := project.Resolve(in.General.Project)
		if err != nil {
			return err
		}
		local, err := paths.ConfigDir()
		if err != nil {
			return err
		}
		c, err := config.Resolve(p.Path, p.Primary, local)
		if err != nil {
			return err
		}
		var entries []provision.HomeEntry
		err = provision.WalkHome(local, p.Path, c.Persistence, func(e provision.HomeEntry) error {
			if ctx.Err() != nil {
				return context.Cause(ctx)
			}
			if e.Relative != "." {
				entries = append(entries, e)
			}
			return nil
		})
		if err != nil {
			return err
		}
		var report strings.Builder
		report.WriteString("Configuration\n=============\n")
		report.Write(config.Format(c))
		report.WriteString("\nHome overlay and persistence\n============================\n")
		report.WriteString("Desired declarative inputs for /home/dev (current configuration).\nInstallers and project setup may create or change additional files.\n\n")
		formatSetupHome(&report, entries, c.Persistence)
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		_, err = out.Write([]byte(report.String()))
		return err
	}}
}
func homeEntrySource(e provision.HomeEntry) string {
	prefix := "home/"
	if e.Layer == "project" {
		prefix = ".vbx/home/"
	}
	return fmt.Sprintf("%s %q", e.Layer, prefix+e.Relative)
}
func formatSetupHome(out *strings.Builder, entries []provision.HomeEntry, paths []config.PersistencePath) {
	type item struct {
		entry       provision.HomeEntry
		previous    []provision.HomeEntry
		persistence *config.PersistencePath
		managed     bool
	}
	items := map[string]item{}
	for _, entry := range entries {
		prior, ok := items[entry.Relative]
		next := item{entry: entry}
		if ok {
			next.previous = append(prior.previous, prior.entry)
		}
		items[entry.Relative] = next
	}
	if len(entries) == 0 {
		out.WriteString("No home overlay entries.\n\n")
	}
	items[".config/vbx/environment.sh"] = item{managed: true}
	for _, p := range paths {
		items[strings.TrimPrefix(p.Guest, "~/")] = item{persistence: &p}
	}
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		v := items[name]
		guest := path.Join("/home/dev", name)
		if v.managed {
			fmt.Fprintf(out, "%q [managed file -rw-------]\n  source: merged envFiles (values hidden)\n", guest)
			continue
		}
		if p := v.persistence; p != nil {
			root := "/mnt/vbx-config-data"
			source := "configuration persistence"
			if p.Module != "" {
				root = "/mnt/vbx-data/" + p.Module
				source = fmt.Sprintf("module %q persistence", p.Module)
			}
			fmt.Fprintf(out, "%q -> %q [persistence link]\n  source: %s\n", guest, path.Join(root, p.Data), source)
			continue
		}
		entry := v.entry
		kind := "file"
		if entry.Info.IsDir() {
			kind = "directory"
		} else if entry.Info.Mode()&os.ModeSymlink != 0 {
			kind = "symlink"
		}
		fmt.Fprintf(out, "%q", guest)
		if kind == "symlink" {
			fmt.Fprintf(out, " -> %q", entry.Link)
		}
		fmt.Fprintf(out, " [%s %s]\n  source: %s\n", kind, entry.Info.Mode().String(), homeEntrySource(entry))
		verb := "overrides"
		if entry.Info.IsDir() {
			verb = "merges"
		}
		for _, old := range v.previous {
			fmt.Fprintf(out, "  %s: %s\n", verb, homeEntrySource(old))
		}
	}
}
