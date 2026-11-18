# Configuration

On this page: [Global example](#example-global-configuration) ·
[Project example](#example-project-configuration) · [Settings](#supported-settings) ·
[Precedence](#files-and-precedence) · [Inspect](#inspect-effective-configuration) ·
[Validation](#validation-during-creation) · [Apply changes](#reuse-and-configuration-changes) ·
[Managed files](#files-vbx-manages)

## Guest account

New VMs use account `dev` with home `/home/dev`. Lima keeps its default numeric
UID mapping to the host user for shared-file ownership.

VMs retain their creation-time account and home.

## Files and precedence

All settings merge from least to most important:

1. VBX defaults: 2 CPUs, 4GiB memory, 30GiB disk, VS Code launch.
2. Settings from globally selected VBX modules, in selection order.
3. User global settings under `${XDG_CONFIG_HOME:-$HOME/.config}/vbx/`.
4. Settings from project-selected modules, in selection order.
5. Project config settings under `PROJECT/.vbx/`.

Each source can provide `config.yaml` and `persistence.yaml`. VBX reads the
sidecar first, then inline settings. `config.yaml` requires `version: 1`;
`persistence.yaml` contains only `persistence.paths` and needs no version field.
Module settings use the same schema as user and project settings.

Later explicit scalars override earlier values. Mount entries merge by guest
`target`, and persistence entries merge by exact `guest`; later entries replace
earlier complete entries. Duplicate keys within a file fail. Effective host
mount overlaps and parent/child persistence mappings fail validation.

Module, environment-file, and APT lists append and deduplicate identical entries.
Later environment files override earlier variable assignments. Different APT
constraints remain separate requirements for the combined solver phase. Empty
lists preserve inherited entries. Module scripts run once in first-selection
order. A module selected in both scopes contributes settings at both module
layers.

## Example global configuration

For Git and compiler tools in new VMs, create the global directory:

```sh
mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/vbx"
```

Create `config.yaml` in that directory with the following contents. If it already
exists, add the packages to its `apt.packages` list and keep its other settings.

```yaml
version: 1
apt:
  packages:
    - git
    - build-essential
```

Select tools before first creation. For a ready VM, follow
[applying changes](#reuse-and-configuration-changes).
The checkout's `config.local.example/` also contains a package-selection example,
optional environment keys, and Pi home-file examples. Copy only what you need.

## Example project configuration

Create `.vbx/config.yaml`:

```yaml
version: 1
resources:
  cpus: 4
  memory: 8GiB
  disk: 40GiB
launch:
  default: vscode
  folder: /workspaces
apt:
  packages:
    - build-essential
    - gh
    - postgresql-client
    - 'jq (>= 1.7)'
mounts:
  - source: ../api
    target: /workspaces/api
    readonly: false
  - source: ../reference
    target: /workspaces/reference
    readonly: true
```

Create the source directories before starting VBX. Project-relative sources resolve
from the project root. Global relative sources resolve from
`${XDG_CONFIG_HOME:-$HOME/.config}/vbx/`.
Absolute paths and `~/` are supported. Environment-variable expansion is not performed.

## Supported settings

Defaults below apply before configuration layers merge.

| Setting | Values | Default |
| --- | --- | --- |
| `version` | `1` | Required in `config.yaml` |
| `resources.cpus` | Positive integer | `2` |
| `resources.memory` | Positive integer followed by `MiB`, `GiB`, or `TiB` | `4GiB` |
| `resources.disk` | Positive integer followed by `MiB`, `GiB`, or `TiB` | `30GiB` |
| `launch.default` | `vscode`, `terminal`, or `workspace-only` | `vscode` |
| `launch.folder` | Absolute guest path opened by VS Code | Automatic (see below) |
| `mounts[].source` | Existing host directory | Required per entry |
| `mounts[].target` | Immediate child of `/workspaces`, such as `/workspaces/api` | Required per entry |
| `mounts[].readonly` | `true` or `false` | `false` |
| `modules[]` | Explicitly ordered [bundled modules](recipes.md#selection) | `[]` |
| `apt.packages[]` | Package name, optionally `package (>= VERSION)`, `package (<= VERSION)`, or `package (= VERSION)` | `[]` |
| `envFiles[]` | Existing dotenv-style file; same path rules as mount sources | `[]` |
| `persistence.paths[]` | `guest: ~/DIRECTORY` and `data: RELATIVE_DIRECTORY`; see [persistence](persistence.md) | `[]` |

The primary project is writable at `/workspaces/<project-slug>`. Terminals and
explicit commands start there. VS Code opens that directory when no extra mounts
exist, or `/workspaces` when additional directories are mounted. `launch.folder`
overrides the editor folder only.

New VMs use `cache: loose` on writable 9p workspace mounts so editors can
receive file-change notifications for guest-side edits. Loose caching can
make host-side changes temporarily stale inside the guest. Read-only mounts
and retained-data mounts keep Lima's defaults. Existing VMs keep their current
mount settings.

Use two-space indentation and the block structure shown above. The parser
supports plain scalars, single-quoted strings (escape `'` as `''`), JSON-compatible
double-quoted strings, comments, and empty lists (`mounts: []`, `modules: []`,
`envFiles: []`, or nested `apt:` / `packages: []`). It rejects other YAML constructs,
unknown settings, duplicate keys, and tabs. Configuration is data; it is never
sourced as shell code.
VM snapshots require UTF-8 configuration paths and values.

## Inspect effective configuration

Run `vbx show-project-effective-setup` from the project directory, or use
`vbx show-project-effective-setup --project DIR`. The Configuration section shows
merged settings with defaults filled in and paths resolved. It uses the same
setting names and YAML structure as `config.yaml`. `apt.packages` includes
requirements from selected modules' `apt.txt` files and all configuration layers.
Provisioning captures this same resolved package list.

The Home overlay and persistence section shows the merged home entries, source
metadata, and persistence connections.

## Validation during creation

During VM creation, VBX validates the desired configuration, referenced
file/directory paths, and environment-file contents. Lima validates the generated
VM template before startup. Validation errors stop creation.

Creating a VM with a project `.vbx/` directory requires interactive approval.
APT resolves the combined [package requirements](recipes.md#apt-requirements)
against guest repositories. Conflicts and unavailable versions stop setup;
VBX prevents package removals.
See [modules](recipes.md) for tool selection, [credentials](credentials.md) for
environment files, [provisioning](provisioning.md) for home overlays and setup,
and [troubleshooting](troubleshooting.md) for failures.

## Reuse and configuration changes

Ready VMs reuse their saved configuration. Editing either configuration file,
home overlays, environment-file contents, modules, or setup scripts does not
change a ready VM.
To apply changes to a ready VM:

```sh
vbx remove
vbx start --vscode
```

Removal discards guest-local data. [Back it up first](persistence.md#back-up-guest-local-files).
Failed creation leaves a disposable incomplete
VM for inspection. Correct its setup inputs, then remove and recreate it. Ordinary
retries use the original snapshot.

VBX derives each managed `vbx-` instance name from the canonical project path.
See [project identity](commands.md#what-counts-as-a-project) before using a
subdirectory, moving a project, or switching worktrees.

## Files VBX manages

| Files | Location |
| --- | --- |
| Lima VM data | Lima's configured home (`LIMA_HOME`, normally `~/.lima`) |
| Saved configuration, creation status, and retained setup archive | `${XDG_STATE_HOME:-$HOME/.local/state}/vbx/instances/` |
| Retained module data | `${XDG_DATA_HOME:-$HOME/.local/share}/vbx/projects/<project-id>/modules/<module>/` |
| Configuration-defined retained data | `${XDG_DATA_HOME:-$HOME/.local/share}/vbx/projects/<project-id>/config/` |
| Operation locks | `${XDG_STATE_HOME:-$HOME/.local/state}/vbx/locks/` |
| SSH configuration links | `${XDG_CONFIG_HOME:-$HOME/.config}/vbx/ssh/` |

VBX prints the SSH `Include` line with `vbx show-host-ssh-config-line`. It does not edit
`~/.ssh/config`. Creation and startup are locked per project; the lock is released
before opening the editor, shell, or command so multiple sessions can coexist.
