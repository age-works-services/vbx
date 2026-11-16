// Package cli parses invocations and dispatches through the command contract.
package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"vbx/internal/command"
)

func usage(format string, args ...any) error {
	return &command.UsageError{Message: fmt.Sprintf(format, args...)}
}
func Validate(commands []command.Command) error {
	seen := map[string]bool{}
	for _, c := range commands {
		if c == nil {
			return fmt.Errorf("nil command")
		}
		s := c.Spec()
		if s.Name == "" || strings.ContainsAny(s.Name, "= \t\r\n") || strings.HasPrefix(s.Name, "-") || s.Summary == "" || s.Usage == "" || seen[s.Name] {
			return fmt.Errorf("invalid or duplicate command %q", s.Name)
		}
		seen[s.Name] = true
		if s.Operands > command.RequiredOperands {
			return fmt.Errorf("invalid operand policy for %s", s.Name)
		}
		for name, f := range s.Flags {
			if name == "" || strings.ContainsAny(name, "= \t\r\n") || strings.HasPrefix(name, "-") || name == "project" || name == "help" || name == "h" {
				return fmt.Errorf("invalid or colliding command flag %q", name)
			}
			if f.Kind != command.Boolean {
				return fmt.Errorf("unsupported flag kind for %s", name)
			}
		}
	}
	if !seen["start"] {
		return fmt.Errorf("missing default start command")
	}
	return nil
}
func help(out io.Writer, commands []command.Command, selected command.Command) {
	if selected == nil {
		fmt.Fprintln(out, "Usage: vbx [GENERAL OPTIONS] COMMAND [OPTIONS]\n       vbx COMMAND [GENERAL OPTIONS] [OPTIONS]\n\nWith no command, defaults to start.\n\nCommands:")
		specs := make([]command.Spec, 0, len(commands))
		for _, c := range commands {
			specs = append(specs, c.Spec())
		}
		sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
		width := 14
		for _, s := range specs {
			if len(s.Name) > width {
				width = len(s.Name)
			}
		}
		for _, s := range specs {
			fmt.Fprintf(out, "  %-*s %s\n", width, s.Name, s.Summary)
		}
	} else {
		s := selected.Spec()
		fmt.Fprintf(out, "Usage: vbx %s\n\n%s\n", s.Usage, s.Summary)
		names := []string{}
		for n := range s.Flags {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintf(out, "  --%-16s %s\n", n, s.Flags[n].Summary)
		}
	}
	fmt.Fprintln(out, "\nGeneral options (before or after command):\n  --project DIR       Select project (default: current directory)\n  -h, --help          Show help")
}
func Run(ctx context.Context, args []string, commands []command.Command, out io.Writer) error {
	if err := Validate(commands); err != nil {
		return err
	}
	general := command.GeneralOptions{Project: "."}
	flags := map[string]bool{}
	seen := map[string]bool{}
	var selected command.Command
	var operands []string
	wantHelp := false
	for i := 0; i < len(args); i++ {
		s := args[i]
		if s == "--" {
			if selected == nil {
				return usage("-- requires a command first")
			}
			operands = append(operands, args[i+1:]...)
			break
		}
		if strings.HasPrefix(s, "-") {
			if s != "-h" && !strings.HasPrefix(s, "--") {
				return usage("unknown option: %s", s)
			}
			name, value, hasValue := strings.Cut(strings.TrimPrefix(s, "--"), "=")
			if s == "-h" {
				name = "help"
			}
			if name == "help" || name == "h" {
				if hasValue {
					return usage("help takes no value")
				}
				wantHelp = true
				continue
			}
			if name == "project" {
				if seen[name] {
					return usage("repeated option --project")
				}
				seen[name] = true
				if !hasValue {
					i++
					if i >= len(args) || strings.HasPrefix(args[i], "--") {
						return usage("--project requires a directory (use --project=PATH for dash-leading paths)")
					}
					value = args[i]
				}
				if value == "" {
					return usage("--project requires a directory")
				}
				general.Project = value
				continue
			}
			if selected != nil {
				if _, ok := selected.Spec().Flags[name]; ok {
					if hasValue {
						return usage("--%s takes no value", name)
					}
					if seen[name] {
						return usage("repeated option --%s", name)
					}
					seen[name] = true
					flags[name] = true
					continue
				}
			}
			return usage("unknown option: %s", s)
		}
		if selected == nil {
			for _, c := range commands {
				if c.Spec().Name == s {
					selected = c
					break
				}
			}
			if selected == nil {
				return usage("unknown command: %s", s)
			}
			continue
		}
		operands = append(operands, args[i:]...)
		break
	}
	if wantHelp {
		help(out, commands, selected)
		return nil
	}
	if selected == nil {
		for _, c := range commands {
			if c.Spec().Name == "start" {
				selected = c
				break
			}
		}
	}
	spec := selected.Spec()
	if spec.Operands == command.NoOperands && len(operands) > 0 {
		return usage("unexpected argument: %s", operands[0])
	}
	if spec.Operands == command.RequiredOperands && len(operands) == 0 {
		hint := spec.OperandHint
		if hint == "" {
			hint = "arguments"
		}
		return usage("%s requires %s", spec.Name, hint)
	}
	return selected.Run(ctx, command.NewInvocation(general, flags, operands))
}
