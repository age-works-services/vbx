# Development

## Build and validate

Use Go 1.25 or later. The host application uses the standard library only.

```sh
make help
make build
make test
make check
make cross-check
```

`bin/vbx` embeds public guest scripts, modules, and the Lima template. Rebuild
after editing those assets. Private local/project configuration remains external.
The executable does not need the source checkout at runtime.

Race tests require a C compiler and cgo. Release builds disable cgo. Cross-builds
check compilation only; supported VM hosts remain Linux x86_64 with KVM.

## Install

`make install` builds the standalone executable and installs it with mode 755
to `$HOME/.local/bin/vbx`, creating the destination directory as needed.
Ensure `$HOME/.local/bin` is in `PATH`. Installation only copies the executable;
configuration and VMs remain unchanged.

Override installation paths through Make variables:

```sh
make install PREFIX=/opt/vbx
make install BINDIR=/path/to/bin
make install PREFIX=/usr/local DESTDIR=/tmp/vbx-package
```

`BINDIR` defaults to `$(PREFIX)/bin`. `DESTDIR` prepends a staging root to
`BINDIR`. Choose an absolute installation directory and supply elevated
permissions yourself if the destination requires them.

`make help` lists all targets and installation variables. Bare `make` (or
`make all`) builds, runs the full test suite, and installs in that order.
A build or test failure prevents installation. This ordering also applies with
`make -j`. Installation overrides apply to the default workflow as well.

## Add a command

Create a file in `internal/commands/`, using the same package as existing commands:

```go
package commands

import (
    "context"
    "fmt"

    "vbx/internal/command"
    "vbx/internal/project"
)

func init() { register(newExampleCommand) }

func newExampleCommand(d Dependencies) command.Command {
    out := d.Out
    return command.Handler{
        Definition: command.Spec{
            Name: "example",
            Summary: "Print project identity",
            Usage: "example [--detail]",
            Flags: map[string]command.Flag{
                "detail": {Summary: "Include the canonical project path"},
            },
        },
        Execute: func(ctx context.Context, in command.Invocation) error {
            p, err := project.Resolve(in.General.Project)
            if err != nil {
                return err
            }
            if in.Flags.Bool("detail") {
                _, err = fmt.Fprintf(out, "%s %s\n", p.Name, p.Path)
            } else {
                _, err = fmt.Fprintln(out, p.Name)
            }
            return err
        },
    }
}
```

Registration includes the command in dispatch and generated help. Factories do
no I/O and retain only the capabilities their handlers use. `Dependencies` groups
constructor inputs. The handler owns its validation, sequencing, and output.
Return errors to the caller; the entrypoint owns process exit.

Flag definitions declare their value kind (currently boolean). Operand policies
are `NoOperands`, `OptionalOperands`, and `RequiredOperands`. The CLI validates
syntax and operand policy after handling help. Handlers validate further semantic
constraints and return `command.UsageError` for invalid invocations.

Add tests and user documentation. Handler tests inject narrow service fakes.
Registry tests use `NewWith` with an isolated factory list, so test commands stay
local to their tests. Existing capabilities let ordinary commands grow through a
command file, tests, and documentation. New capabilities extend their owning
service and the composition wiring.

## Boundaries

The [architecture reference](reference/architecture.md) defines ownership and
resource lifetimes in detail.

- `command` defines metadata, copied invocation values, and shared usage errors.
- `cli` owns argument boundaries, general-option scope, selection, and help.
- `commands` owns each user-facing operation, including final launch handoff.
- `workspace` owns shared readiness and lifecycle scope acquisition.
- `backend/lima` owns Lima primitives, template rendering, and managed shell argv.
- `host` checks prerequisites; `interaction` supplies cancellable terminal input.
- `config`, `project`, `state`, and `provision` own configuration, identity, persisted data, paths, locks, and frozen bundles.
- `process` owns OS execution, terminal detection, cancellation, and replacement.
- Guest Bash scripts and modules own installation and functional checks.

Do not read private configuration in tests. Use temporary HOME/XDG paths,
synthetic credentials, and fake system tools. VM changes require separate real-VM
validation described in [validation](testing/validation.md).
