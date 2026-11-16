# Commands

`vbx` defaults to `vbx start`. Both create/start/reuse the project VM and launch
its configured target. Existing VMs use their captured `launch.default`.

For common tasks, see the [quick reference](quick-reference.md). For failures,
see [troubleshooting](troubleshooting.md).

On this page: [Project identity](#what-counts-as-a-project) ·
[Commands and options](#commands-and-options) · [Effective setup](#effective-project-setup) ·
[Guest arguments](#guest-command-arguments) · [Status and inventory](#vm-status-and-inventory) ·
[Removal](#stop-removal-and-recovery) · [SSH](#ssh-access) · [Exit status](#cli-edge-cases-and-exit-status)

## What counts as a project?

VBX uses the exact directory selected by `--project DIR`, or the current directory
when omitted. It does not search upward for a Git root or `.vbx/` directory.
Run from the intended project root for daily use.

- A subdirectory selects a separate VM and its own `.vbx/` configuration.
- Separate worktrees select separate VMs and retained-data stores.
- Symlink paths that resolve to the same directory select the same VM.
- Moving or renaming a directory changes its canonical path and selects a new VM
  and retained-data store. VBX does not migrate the old ones automatically.
- Project selection requires an existing directory other than `/`.

Use `vbx name` to inspect the selected identity and `vbx list-vms` to find older
instances. Before moving a project, back up guest-local files and remove its VM.
For an already moved or deleted project, see [recovery](troubleshooting.md#moved-or-deleted-projects).

## Commands and options

| Command | Behavior |
| --- | --- |
| `start` | Ensure readiness and launch the configured target |
| `stop` | Stop the existing VM |
| `status` | Show the selected project's current VM status |
| `list-vms` | List all known VBX VMs with project paths and status |
| `remove` | Confirm and delete the VM and its VBX state, preserving retained project data |
| `remove-project-data [PROJECT-ID]` | Confirm and delete retained project data after the VM has been removed |
| `exec COMMAND [ARG...]` | Ensure readiness and execute the guest command |
| `name` | Print the canonical project VM name |
| `show-project-effective-setup` | Inspect merged configuration, home overlay metadata, and persistence connections |
| `show-host-ssh-config-line` | Print the host SSH Include line (does not edit SSH configuration) |
| `check-host-prerequisites` | Check Linux host prerequisites |

General options work before or after the command:

```sh
vbx --project /path/to/project start --vscode
vbx start --vscode --project /path/to/project
vbx remove --project /path/to/project
```

- `--project DIR` selects the project (default: current directory).
- `--project=DIR` also works. Relative paths resolve from the invocation directory.
- `-h` and `--help` show general or selected-command help without accessing a VM.
- Options use double dashes, except `-h`. Boolean options take no explicit value.
- For `exec`, `--` explicitly begins guest argv (see below).

See [CLI edge cases and exit status](#cli-edge-cases-and-exit-status) for scripting details.

`start` accepts one optional override, after the command name:

```sh
vbx start --vscode
vbx start --terminal
vbx start --workspace-only
```

These options are mutually exclusive and do not modify saved configuration.
`--workspace-only` starts without opening a tool. No agent starts automatically.

## Effective project setup

```sh
vbx show-project-effective-setup
vbx show-project-effective-setup --project /path/to/project
```

The report has two sections:

- **Configuration** contains merged settings with defaults, resolved paths,
  resources, launch settings, mounts, module order, combined APT requirements,
  environment-file paths, and persistence mappings.
- **Home overlay and persistence** lists the effective global/project overlay
  entries beneath `/home/dev`, their types and modes, winning relative source
  filenames, and overridden sources. Directories merge recursively. Symbolic
  links show their literal targets. Persistence entries show their expected
  guest targets and storage owners. The managed environment file appears with
  its values hidden.

The command reads home metadata without opening file contents or following child
symlinks. It uses the same overlay and persistence collision checks as creation.
File/directory conflicts fail before the report is printed. It runs offline and
does not contact a VM, execute setup, allocate storage, or read environment values.

The report describes current desired declarative inputs. Module installers and
project scripts can create or change additional files. Existing VMs keep their
captured setup until explicit recreation.

## Guest command arguments

VBX parses its options until the first guest command operand or an explicit `--`:

```sh
vbx --project /path/to/project exec npm test
vbx exec --project /path/to/project -- npm test -- --watch
vbx exec printf '%s\n' 'a value with spaces' '--project'
vbx exec -- bash -c 'make && ./app'
```

All arguments after the guest executable belong to that executable.
`vbx exec npm --project other` forwards `--project other` to npm. Use
`vbx exec --help` for VBX help and `vbx exec npm --help` for npm help.

Arguments retain their boundaries and never become host shell code. Use an
explicit guest shell for pipelines or multiple commands. Managed commands load
the captured guest environment, prepend `~/.local/bin` to PATH, and run from the
primary project mount under `/workspaces/<project-slug>`.

## VM status and inventory

```sh
vbx status
vbx status --project /path/to/project
vbx --project /path/to/project status
vbx list-vms
```

`status` prints the canonical VM name and its current Lima status (for example,
`Running` or `Stopped`). It prints `Missing` if Lima has no VM for that project.
A missing VM is a successful status query. Invalid project paths and failed Lima
queries produce operational errors.

`list-vms` prints a table with `NAME`, `PROJECT`, and `STATUS` columns, sorted by VM
name. It includes Lima instances in the `vbx-` namespace and excludes other Lima
VMs. Project paths come from the captured primary workspace mount, so deleted
project directories and VMs without VBX state can still appear. A dash indicates
that the project mount is unavailable. An empty inventory prints just the header.

Both commands query Lima without starting, stopping, provisioning, locking, or
repairing a VM. They do not load desired project/user configuration. `list-vms` is
global, so `--project` does not filter it. Saved VBX state without a corresponding
Lima VM does not appear in `list-vms`. These commands report Lima's runtime status,
which is separate from provisioning completion.

## Stop, removal, and recovery

```sh
vbx stop
vbx remove
```

`remove-project-data` deletes the current project's retained data, or
accepts an explicit project identifier for a moved/deleted project. It requires
the VM to be absent and confirmation by typing the identifier. An explicit
identifier selects the target; `--project DIR` selects it when the operand is
omitted. See [project module persistence](persistence.md).

Stopping preserves installed tools and guest-local files. Removal requires a real
interactive terminal and confirmation by typing the VM name. It deletes VBX state
only after Lima deletion succeeds. Mounted host files remain intact.

If a VM is missing but VBX instance state remains, startup reports the mismatch
without automatically repairing or replacing it. You decide how to recover.
`vbx remove` can explicitly discard orphaned instance state after confirmation.
A missing or invalid snapshot for an existing VM also stops startup safely.

Before removal, [back up guest-local files](persistence.md#back-up-guest-local-files).
For failed setup, state mismatches, and moved/deleted projects, see
[troubleshooting](troubleshooting.md).
For changed configuration, follow [recreation](configuration.md#reuse-and-configuration-changes).

## SSH access

Run `vbx show-host-ssh-config-line` and add the printed `Include` near the
beginning of `~/.ssh/config`. Start the VM, then connect with OpenSSH:

```sh
vbx start --workspace-only
ssh "lima-$(vbx name)"
```

Other SSH clients must read this OpenSSH configuration or import the connection
settings it references. For host-agent forwarding and private repositories, see
[Git authentication](credentials.md#git-authentication).
Direct SSH sessions do not run VBX readiness checks.
For a failed guest, use Lima directly:

```sh
limactl shell --workdir=/ "$(vbx name)"
```

To forward a guest service on port 3000 to an explicit host loopback port:

```sh
ssh -N -L 127.0.0.1:13000:127.0.0.1:3000 "lima-$(vbx name)"
```

Keep that SSH session open. See [networking](installation.md#networking) for scope.

## CLI edge cases and exit status

- Repeated value/launch options, unknown arguments, and surplus operands fail.
- Help suppresses command argument validation, but malformed or unknown options
  and unknown commands still produce usage errors.
- `--` requires a command first.
- CLI usage errors exit 2. Operational failures exit 1. Host SIGHUP/SIGINT/SIGTERM
  exit 129/130/143 after cleanup. Final guest handoff preserves process status.
