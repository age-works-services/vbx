# Project persistence and backup

On this page: [What survives](#what-survives) · [Retain tool state](#retain-tool-state) ·
[Custom paths](#configuration-defined-paths) · [Storage](#storage-and-lifetime) ·
[Backup](#back-up-guest-local-files) · [Conflicts](#conflicts-and-existing-data) ·
[Delete data](#explicit-data-deletion)

## What survives?

| Data | After `vbx stop` | After `vbx remove` |
| --- | --- | --- |
| Host project files and additional host mounts | Kept | Kept |
| Declared retained directories | Kept | Kept |
| Other guest-local files and installed tools | Kept | Deleted |

Removal does not undo edits guest programs made to host files. Use your normal
host backup process for project files and retained data.

## Retain tool state

Add modules to global or project `config.yaml` (which also requires `version: 1`).
Select persistence modules to retain working state across VM removal and
recreation for the same project path:

```yaml
modules:
  - node
  - pi
  - pi-project-persistent-sessions
  - vscode-project-persistent-state
```

Each module has one purpose. The Pi installer installs Pi; its separate
persistence module retains sessions. The VS Code persistence module retains
remote extension installations and remote machine settings together.

| Module | Guest directory | Module data subdirectory |
| --- | --- | --- |
| `pi-project-persistent-sessions` | `~/.pi/agent/sessions` | `sessions` |
| `vscode-project-persistent-state` | `~/.vscode-server/extensions` | `extensions` |
| `vscode-project-persistent-state` | `~/.vscode-server/data/Machine` | `machine-settings` |

## Configuration-defined paths

Use the same structure inside user or project `config.yaml`, or in a sibling
`persistence.yaml`:

```yaml
persistence:
  paths:
    - guest: ~/.local/share/my-tool
      data: my-tool
```

A `config.yaml` also needs `version: 1`. The sidecar contains only the
`persistence` block. Within each source, inline declarations override the sidecar.

Declarations follow the [configuration layer order](configuration.md#files-and-precedence)
and merge by exact `guest` path. The winning entry selects its `data` path and
storage scope.

Duplicate guest keys within one file fail. After merging, parent/child guest
paths and overlapping data paths within a storage scope fail validation.
`show-project-effective-setup` includes the effective `persistence.paths` set.

## Storage and lifetime

Every selected module receives its own private directory:

```text
${XDG_DATA_HOME:-$HOME/.local/share}/vbx/projects/<project-id>/modules/<module>/
```

VBX mounts it at `/mnt/vbx-data/<module>` and connects declared guest directories
using symbolic links. Configuration-defined mappings share a separate store:

```text
${XDG_DATA_HOME:-$HOME/.local/share}/vbx/projects/<project-id>/config/
```

VBX mounts this store at `/mnt/vbx-config-data`. Global and project configuration
use this same project-scoped store. Overriding a module declaration with a
configuration declaration changes its storage scope. VBX preserves the former
store; transfer valuable contents explicitly before switching stores.

`XDG_DATA_HOME` must be absolute. VBX-owned data-directory
levels must be real directories owned by the host user with mode 0700.

The project identifier is the VM name printed by `vbx name`. Recreating the VM
for the same canonical project path and data-home base reconnects retained data.
Changing either selects a different storage location. Existing VMs retain their
captured absolute bindings.

`vbx remove` preserves all retained project data. Removing a module from configuration
also preserves its stored contents. Session histories and settings can contain
sensitive information; keep the host data directory private.

## Back up guest-local files

Before removal, identify valuable files outside host mounts and declared retained
directories. Stop editors, agents, and other writers while copying their state.
Keep the VM running for the copy.

For example, archive a guest-local tool directory to a new private host directory:

```sh
name="$(vbx name)"
backup="$(mktemp -d "$HOME/vbx-backup.XXXXXX")"
limactl shell --workdir=/ "$name" -- \
    tar -C /home/dev -czf - .local/share/my-tool > "$backup/my-tool.tar.gz"
```

Replace `.local/share/my-tool` with the path you need, relative to `/home/dev`.
Check that the command succeeds, inspect the archive with
`tar -tzf "$backup/my-tool.tar.gz"`, and verify important contents before removal.
A failed copy can leave a partial archive. Protect backups that contain secrets.

This direct Lima command also works when VBX provisioning has failed, provided
the guest is running and accessible. If stopped, try `limactl start "$name"`;
this starts Lima directly without retrying VBX setup. For a missing project path,
follow [recovery](troubleshooting.md#moved-or-deleted-projects) first.

To restore after recreation, extract selected files into a temporary directory
inside the host project mount, then copy them to their intended guest locations.
Avoid overwriting VBX-managed files or persistence links.

## Conflicts and existing data

Persistence is configured during VM creation. Preserve valuable guest-local
state before removing an existing VM to enable persistence.

Guest directories must be absent, empty, or already the expected symbolic links.
VBX reports populated directories and conflicting links for explicit resolution.
Inspect a blocked VM directly:

```sh
limactl shell --workdir=/ "$(vbx name)"
```

Copy valuable data to a safe location before changing either copy or removing
the VM. Replacing a link with a populated guest-local directory causes readiness
to fail, preserving that directory for recovery.

Home overlays must keep entries outside declared persistent directories. VBX
rejects overlapping overlay entries before starting the VM. Persistent mappings
are applied after overlays and the managed environment, before project setup.
Retries preserve existing links and reapply the captured mappings.

## Explicit data deletion

First remove the VM. Then delete retained data for the current project:

```sh
vbx remove
vbx remove-project-data
```

Select another project with `--project DIR`. If its directory was moved or
deleted, supply its saved identifier directly:

```sh
vbx remove-project-data vbx-project-123456abcdef
```

An explicit identifier selects the deletion target. The operation requires a real
terminal and confirmation by typing that identifier. The VM must be absent;
a stopped VM still retains its bindings. Deletion removes the selected project's
entire retained-data tree (module and configuration-defined stores) under the
current data-home base.

For module declarations and installer contracts, see
[developing bundled modules](../module-development.md#persistence-declarations).
