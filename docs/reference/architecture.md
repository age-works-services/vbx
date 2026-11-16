# Application architecture

VBX is a Go host application with Bash guest provisioning. The host executable
embeds the public VM template, guest scripts, and bundled modules. User and
project configuration remain external.

Each command owns its behavior and orchestration in its related Go file. Shared
services provide reusable workspace operations, backend primitives, and data
handling. The entrypoint composes these capabilities, and the CLI connects parsed
invocations to commands.

## Package structure

```text
cmd/vbx/main.go              Process entrypoint and dependency composition
internal/
  command/                  Command contract and invocation data
  cli/                      Argument parsing, help, and command selection
  commands/                 One implementation file per command
    registry.go             Factory registration and construction
    start.go
    stop.go
    status.go
    list_vms.go
    remove.go
    remove_project_data.go
    exec.go
    name.go
    show_project_effective_setup.go
    show_host_ssh_config_line.go
    check_host_prerequisites.go
  workspace/                Shared workspace readiness workflow
    ready.go
    scope.go                Lifecycle scope acquisition and release
    create.go
    provision.go
  backend/lima/             Lima primitives and template rendering
  host/                     Supported-platform and prerequisite checks
  config/                   Desired configuration parsing and merging
  project/                  Canonical project identity and primary mount path
  state/                    Snapshots, status, paths, and lifecycle locks
  provision/                Frozen bundle, dotenv, and APT input construction
  module/                   Module definitions and persistence-path validation
  process/                  Subprocess execution, signals, and final handoff
  interaction/              Cancellable terminal input
assets.go                   Embedded public assets
scripts/                    Guest Bash provisioning
modules/                    Self-contained guest module installers and assets
```

Files split along these responsibilities as the application grows. Each package
has a focused API and owns the resources associated with its operations.

## Entry point and dependency composition

`cmd/vbx/main.go` assembles the concrete process runner, streams, embedded asset
filesystem, terminal interaction, Lima client, host checks, and workspace service.
It builds the command collection and passes it to the CLI.

The entrypoint owns signal registration and exit-code translation. Commands own
user-facing operations, and services own their reusable implementation primitives.

Service construction establishes dependencies. Configuration reads, project
resolution, host probes, locks, and Lima queries happen when a command requests
the corresponding operation. Help uses command metadata alone.

Interfaces sit at side-effect boundaries and belong to their consumers:

- `commands` declares small interfaces for readiness, scope acquisition, Lima
  operations, process replacement, and line input. Each command constructor
  retains the specific capabilities its handler uses.
- `workspace` declares the Lima, host-check, and interaction interfaces used by
  its readiness workflow.
- `backend/lima` provides a concrete client over `process.Runner`, which supplies
  the subprocess-argv test seam.

The `commands.Dependencies` constructor input holds these interface values,
streams, and paths. Pure configuration, identity, and rendering functions
use concrete types.

## Command contract

`internal/command` defines the contract shared by commands and the CLI. Its
imports cover contract types and standard-library facilities.

```go
type Command interface {
    Spec() Spec
    Run(context.Context, Invocation) error
}

type Invocation struct {
    General GeneralOptions
    Flags   FlagValues
    Args    []string
}

type GeneralOptions struct {
    Project string
}
```

- `Spec` declares the command name, summary, usage, flag definitions, and operand
  policy. The CLI uses this metadata for parsing, validation, and generated help.
- Flag definitions declare their value kind. Command flags initially use the
  boolean kind. `FlagValues` provides validated, typed access to supplied options.
  The contract package provides constructors for invocation values used by the
  parser and tests; constructors copy mutable input collections.
- Operand policy declares whether a command accepts zero operands, optional
  operands, or required operands. Commands apply further semantic constraints.
- The parser validates syntax, option scope, unknown arguments, and duplicate
  options. Commands validate constraints such as mutually exclusive launch targets.
- `Run` receives invocation data and a cancellation context. It returns success
  or an error to the entrypoint through the CLI.
- The shared usage-error type belongs to `command`. Both the parser and handlers
  use it for argument-validation errors.

Each invocation owns its parsed values and argument slice. The parser supplies
`.` as the default project path and completes construction before execution.
Handlers treat invocation data as read-only.
Each handler retains the injected capabilities it uses. `command.Handler` adapts
command-local execution functions to the contract. `context.Context` carries
cancellation and deadlines.

## Registration and adding commands

Each ordinary command source file belongs to `internal/commands` and registers a
factory during package initialization:

```go
func init() {
    register(newRemoveCommand)
}
```

`registry.go` stores factories and constructs fresh commands from injected
dependencies. The constructor-input `Dependencies` struct lives in `commands`
and groups capabilities for wiring. Per-invocation data belongs to `Invocation`;
workspace resources belong to the service call that acquires them.

Factory registration and command construction are metadata-only operations.
Tests can supply an isolated factory list. The CLI validates the resulting
command collection against its general-option definitions: command names are
unique, flags have valid scopes, and the default `start` command exists. It sorts
command metadata by name when generating help.

Adding a command using existing capabilities consists of its implementation
file, tests, and user documentation. Registration includes it automatically in
dispatch and help. New capabilities extend the appropriate lower service with
operation-oriented methods. Go package compilation supplies the command files.

## CLI responsibilities

`internal/cli` receives a collection of `command.Command` values and depends on
the command contract. Its execution sequence is:

1. Parse general options before the command.
2. Select the named command, defaulting to `start` when the command is omitted.
3. Parse general and declared command options after the command.
4. Stop option parsing at the first operand or an explicit `--`.
5. If help was requested, print help and return.
6. Validate the declared operand policy, construct the invocation, and invoke
   the selected command.

Both forms work:

```text
vbx [GENERAL OPTIONS] COMMAND [COMMAND OPTIONS]
vbx COMMAND [GENERAL OPTIONS AND COMMAND OPTIONS]
```

General options can occur on either side. Each value option and each command
option occurs at most once per invocation. Repeated help requests select the
same help output. Command-specific options follow the command. Values that
happen to match command names retain their role as values. Syntax and scope
errors remain errors; help takes precedence over operand and handler validation.

For `exec`, operands starting with the guest executable form an opaque argument
slice. The process layer passes that slice through to the guest with argument
boundaries preserved. Bare `vbx` and explicit `vbx start` select the same command
and default resolution. Detailed syntax belongs in the
[command manual](../manuals/commands.md).

## Command-owned orchestration

Each command file owns the sequence that fulfills its user-facing operation.

| Command file | Behavior it owns |
| --- | --- |
| `start.go` | Validate launch overrides, ensure readiness, resolve the saved default, launch the selected target or print readiness |
| `exec.go` | Validate guest argv, ensure readiness, invoke the managed guest command |
| `stop.go` | Acquire the lifecycle scope, require an existing VM, stop it |
| `status.go` | Resolve project identity, query Lima inventory, print its VM status or `Missing` |
| `list_vms.go` | Query Lima inventory, select VBX VMs, derive captured project paths, sort and print the table |
| `remove.go` | Acquire the lifecycle scope, confirm the exact name, check VM status, stop/delete in order, remove VBX state and SSH link after success |
| `remove_project_data.go` | Resolve project identity, lock by identifier, require the VM to be absent, confirm the identifier, delete the project's retained data |
| `name.go` | Resolve canonical project identity and print its name |
| `show_project_effective_setup.go` | Resolve desired configuration, validate home overlays, format configuration and effective home metadata with persistence targets |
| `show_host_ssh_config_line.go` | Resolve the SSH include path, escape it, print the directive |
| `check_host_prerequisites.go` | Check host prerequisites, query tool versions, print the result |

Handlers express command policy and operation ordering directly. They call shared
services for reusable capabilities. A small operation has a correspondingly small
handler; a command such as `remove` visibly owns its confirmation and destructive
operation sequence.

## Shared workspace readiness

`start` and `exec` both require a ready workspace. `workspace.EnsureReady` owns
that shared workflow. Its input identifies the project. Its result contains the
ready workspace's identity and saved configuration. The caller uses that result
to choose a tool or execute guest argv.

The workflow:

1. Check host prerequisites needed for startup.
2. Acquire the project's lifecycle scope through `workspace.Acquire`.
3. For an existing VM, validate its snapshot and status and start it using the
   captured configuration.
4. For a new VM, check state consistency, resolve desired configuration, obtain
   project-setup approval, resolve selected modules and their persistent-data
   bindings, validate the rendered template, and capture all setup inputs
   privately before starting the guest.
5. Run pending provisioning from captured inputs and update completion status.
6. Register the SSH configuration link.
7. Remove temporary staging resources, release the lock, and return readiness.

`create.go` and `provision.go` hold the distinct creation and setup operations.
`ready.go` coordinates them. `start.go` owns the subsequent launch-target decision,
and `exec.go` owns the subsequent guest-command invocation.

`workspace/scope.go` owns `Acquire(ctx, projectDir) (*Scope, error)`.
`EnsureReady`, `stop`, and `remove` use this same acquisition operation. It resolves
canonical identity and instance/SSH paths, acquires the per-project lock, and
queries VM and state-directory existence while holding that lock.

`Scope` contains project identity, the instance directory, SSH-link path,
`VMExists`, `StateExists`, and an owned release operation. The separate existence
values let `remove.go` handle orphaned state explicitly. Scope acquisition reads
lifecycle metadata; callers choose their subsequent operations.

The scope owns its lock and provides an idempotent release operation. Acquisition
cleans up partial resources on failure. Callers guarantee release on every return
path. Lifecycle commands can defer release; readiness completes cleanup before
returning to callers that may replace the process.

## Persistent module data contract

Each module has one purpose. Tool installation and working-state persistence
belong to separately selected modules. `pi` owns Pi installation;
`pi-project-persistent-sessions` owns retained project sessions and connects Pi
to that storage. VBX owns project identity, persistent-directory allocation,
mounts, script ordering, and retention. Each persistence module declares which
guest directories persist and their layout under its data directory.

For example:

```yaml
modules:
  - node
  - pi
  - pi-project-persistent-sessions
```

Users select prerequisites explicitly. Each selected module receives the same
project-scoped storage interface; selecting the persistence module applies the
session-persistence behavior.

### Module interface

```text
modules/<name>/
  config.yaml       Optional module settings using the configuration schema
  apt.txt           Optional APT requirements
  setup.sh          Optional installation entrypoint
  persistence.yaml  Optional persistence settings under persistence.paths
  ...               Module-owned helpers, policies, and assets
```

A module supplies settings, an installation entrypoint, or both. Its files serve
one purpose. An installation module uses `setup.sh`; a persistence module can
consist entirely of `persistence.yaml`. Module settings follow the precedence
rules in [Configuration precedence](#configuration-precedence).
`internal/module` validates module definitions
and provides them to workspace creation and bundle building. Definitions,
installation scripts, and helpers are captured together with the selected module
files.

VBX invokes module scripts with these environment variables:

| Variable | Value |
| --- | --- |
| `VBX_MODULE_DIR` | Absolute guest path to the module's captured files during provisioning; modules copy runtime assets to their installed locations |
| `VBX_MODULE_DATA_DIR` | Absolute guest path to the selected module's persistent project directory |

VBX establishes these variables separately for each installation-script
invocation. Scripts resolve their own files through `VBX_MODULE_DIR`, whose
captured bundle is removed after successful provisioning. Every selected module
receives an active host-backed mount at its standard data path, accessible and
writable by the guest user, before installation or persistence application.
Guest-side mount inspection verifies the expected target, host-backed filesystem
type, and Lima mount source associated with the binding. For 9p, the guest-visible
source is the Lima mount tag. A writable VM-local directory alone fails this
check. Readiness performs mount verification on every startup, including
ready-VM reuse. Verification, installation, and persistence-application failures
block launch. Installation scripts tolerate partial execution and previously
retained module data.

### Declarative persistence

Configuration files and `persistence.yaml` use the same YAML path:
`persistence.paths`. Users can declare project-specific mappings directly in
`PROJECT/.vbx/config.yaml` or alongside it in `PROJECT/.vbx/persistence.yaml`.
The user configuration directory supports the same two files.

`pi-project-persistent-sessions/persistence.yaml` declares:

```yaml
persistence:
  paths:
    - guest: ~/.pi/agent/sessions
      data: sessions
```

`vscode-project-persistent-state/persistence.yaml` declares:

```yaml
persistence:
  paths:
    - guest: ~/.vscode-server/extensions
      data: extensions
    - guest: ~/.vscode-server/data/Machine
      data: machine-settings
```

Each entry connects a guest directory to a retained data subdirectory using a
symbolic link. Module declarations use their owning module's data directory.
Declarations from user or project configuration use a separate project-scoped
configuration-data directory. The winning declaration determines both the data
path and its storage scope. Overriding a mapping preserves the former store and
requires explicit transfer if its contents should move to the new store.

VBX merges declarations by exact `guest` path using the five configuration layers
below. A later declaration replaces the entire earlier entry for that guest path.
It then validates and captures the effective set, and applies each mapping once.

The manifest has a nonempty `persistence.paths` list. Each entry requires exactly the string
fields `guest` and `data`. The parser rejects unknown fields, duplicate keys,
and malformed entries. `guest` starts with `~/`, has at least one following
path component, and resolves beneath the guest user's home. `data` is a relative
path beneath the declaring source's data mount (`VBX_MODULE_DATA_DIR` for a
module). Both paths use nonempty components and reject `.` and `..` components,
control characters, and variable expansion. Duplicate guest keys within one
file fail validation. Effective guest mappings must be distinct and
non-overlapping after precedence resolution. Parent/child guest paths remain a
conflict. Effective data mappings within each storage scope must also be
non-overlapping.
Guest mappings must not equal, contain, or lie beneath `~/.config/vbx` or
`~/.local/state/vbx`.

When capturing inputs, VBX rejects local or project home-overlay entries at or
beneath a mapped guest directory, and non-directory ancestors of that directory.
The same overlay checks protect `~/.config/vbx/environment.sh` and
`~/.local/state/vbx`. Collision checks use each captured entry's own type, with
symlinks treated as non-directories. Checks cover the combined local-then-project
overlay tree and reject entry paths passing through an overlay-supplied symlink.
Validation completes before starting the VM, preserving both the overlay inputs
and retained contents for explicit user resolution.

VBX records validated mappings alongside mount bindings in the instance snapshot.
`state` owns these binding and mapping record types. The bundle's mapping list is
generated from the same snapshot record. Existing instances use their captured
mappings. Ready verification runs a guest check embedded in the current
executable, supplied with the snapshot records, after the captured bundle has
been removed.

VBX creates persistent target directories with private permissions. Path
validation uses filesystem checks as well as lexical checks: persistent targets
and their parents remain real directories under the declaring source's data mount.
Each existing ancestor between the guest home and the mapped guest path must be
a real directory owned by the guest user. Missing ancestors are created as
directories. Conflicting symlinks or
file types produce an actionable error before replacing any contents.

A guest mapping can be applied when its path is absent, an empty directory, or
already the correct symbolic link. A correct link has an exact `readlink` match
to the captured absolute target under the winning declaration's data mount. Existing populated
guest directories and conflicting links stop provisioning. Retained contents
stay intact. Installing a link must tolerate interruption and retry; a retry
accepts an already-correct link.

Previously established links remain connected during retries. Links first
appear after every module installer has its completion marker, so retries skip
installation when links are attached. Captured overlays have already passed
mapping-collision validation. VBX reapplies the captured
mappings after overlays and environment installation, accepting correct links.
Ready-VM reuse verifies that every captured mapping has the expected exact link
target before allowing launch. Each target must also be an existing real
directory beneath its verified data mount.

Mapping-conflict and ready-verification errors identify the guest path, retained
data path, and affected VM. They provide `limactl shell --workdir=/ <vm-name>` for
guest access while readiness is blocked, and direct the user to preserve and
resolve both locations. Any removal advice must warn that removal deletes the
guest-local copy.

### Storage identity and paths

```text
Host module data:
${XDG_DATA_HOME:-$HOME/.local/share}/vbx/projects/<project-id>/modules/<module>/

Host configuration-defined data:
${XDG_DATA_HOME:-$HOME/.local/share}/vbx/projects/<project-id>/config/

Guest module data:
/mnt/vbx-data/<module>/

Guest configuration-defined data:
/mnt/vbx-config-data/
```

`<project-id>` is the canonical-path-derived VM name, and `<module>` is its
validated registry identifier. Recreating a VM for the same canonical project
path with the same data-home base reconnects the same module directories. Changing the canonical project
path changes the project identity.

`state.Paths` owns `DataDir` and per-project module-data path resolution alongside
its configuration and lifecycle-state paths. `XDG_DATA_HOME`, when supplied,
must be absolute. Configuration remains under `XDG_CONFIG_HOME`; durable module
data lives under `XDG_DATA_HOME`; disposable instance metadata and retained setup
archives live under `XDG_STATE_HOME`. Recreation and retained-data deletion
resolve their project-data paths under the current data-home base; existing VMs
continue to use their captured absolute bindings.

Workspace creation allocates private host directories under the lifecycle lock.
The instance snapshot records bindings separately from the desired `Config`,
with each binding containing its storage scope (module identifier or
configuration-defined data), absolute host path, absolute guest path, and writable flag. The expected Lima mount configuration
is recorded with the binding. Each selected module has a binding. Effective
configuration-defined mappings add a configuration-data binding. A project with
neither selected modules nor persistence mappings has an empty binding list.
The snapshot also holds the effective validated persistence mappings and their
storage scopes. The Lima renderer includes the bindings as
writable mounts. Guest writes use the host-mapped user identity.

VBX resolves the data-home base to an absolute canonical path. Each VBX-owned
level (`vbx`, `projects`, project identifier, `modules`, module identifier, and
`config`, where applicable)
must be a real directory owned by the invoking user with mode 0700. Existing
symlinks, incompatible types, ownership, or permissions produce an error. Mount
source-overlap validation covers the primary project, additional user mounts,
and persistent data directories together.

Existing VMs use their captured mount bindings. Startup validates their host
sources and compares the instance's Lima mount configuration with the captured
locations, guest mount points, and writable flags before starting the guest.
The Lima backend derives the expected guest mount sources from that validated
configuration. Readiness verifies the active guest mounts before returning. Missing or inaccessible storage produces an error. Mount
changes take effect through explicit VM recreation. Installation and persistence
application run during creation and retries.

### Provisioning order

1. Start the VM with the captured persistent-directory mounts available.
2. Complete base guest preparation and combined APT installation.
3. Run available module `setup.sh` scripts in selection order.
4. Apply the captured local and project home overlays.
5. Install and load the managed environment.
6. Apply the captured effective `persistence.paths` mappings after precedence resolution.
7. Run project `.vbx/setup.sh` from the primary project mount.
8. Record completion, remove disposable setup inputs, and allow launch.

`setup.sh` has a per-module completion marker. Overlays, the managed environment,
and persistence application run on every provisioning attempt. Retries use the
captured module definitions and scripts. Persistence application follows overlays
and precedes project setup so project commands use the retained state.

`pi-project-persistent-sessions` persists Pi's session directory. Pi resumes
sessions for the same guest working directory, which stays stable for the same
canonical project path. The VS Code persistence module declares both remote
settings and extension locations before the first client connection installs
the remote server. The parent server directory and
server binaries can already exist; the mapped directories themselves must be
absent, empty, or the correct links. Persistence modules declare data
locations whose file I/O, locking, and file-watching requirements work with the
configured mount type. Validate extension installation, extension updates, remote
settings, and Pi session resumption through VM recreation on that mount type.

### Retention and access

- Stopping, removing, or recreating a VM preserves its project module data.
- Instance-state and SSH-link cleanup operate on their respective paths and
  preserve the separate project-data tree, including after failed provisioning.
- Removing a module from configuration retains its existing data directory.
- A retained-data deletion command owns confirmation and deletion of one
  project's entire module-data tree. It accepts a project path or validated
  project identifier so data remains addressable after a project is moved or
  deleted. Identifiers must match `^vbx-[a-z0-9][a-z0-9-]{0,23}-[0-9a-f]{12}$`,
  the canonical VM-name format. Confirmation requires the project identifier.
- Retained-data deletion acquires the lifecycle lock by project identifier using
  the ID-keyed `state.Lock` primitive. This path remains usable when project-path
  scope acquisition cannot resolve a moved or deleted directory. Deletion
  requires the owning VM to be absent. A VM's stopped status retains its mount
  bindings and therefore does not satisfy this requirement. The command removes
  only the selected project's data tree after these checks succeed.
- Module data can contain credentials and session history. Host directories use
  private permissions, and diagnostics preserve content confidentiality.
- Provisioning and guest programs can write mounted module data. The lifecycle
  lock serializes VBX operations; applications own coordination of live writes.
- Modules receive bindings for their own data. Guest programs retain the same
  trusted-code access model as other writable mounts in the VM.

## Supporting services

- `backend/lima` exposes structured VM inventory (including captured mounts),
  existence/status queries, start, stop, delete, and SSH
  metadata retrieval. It passes explicit argv to `process`. Its `Delete` primitive
  performs Lima deletion; `remove.go` owns the surrounding confirmation, stop,
  and VBX-state cleanup sequence.
- `backend/lima` also owns pure `ShellArgv(instance, workdir, argv)` and
  `SSHHost(instance)` helpers. `ShellArgv` builds the managed shell invocation,
  including the fixed guest-environment wrapper and literal argv forwarding.
  `SSHHost` returns Lima's SSH alias. `start.go` uses these for terminal and VS Code
  launches, and `exec.go` uses them for guest commands. Commands perform the final
  handoff through their injected process replacer.
- Template rendering is a pure operation used during workspace creation. It
  lives with the Lima format.
- `host` checks supported platform, required tools, and virtualization access.
  `check_host_prerequisites.go` owns version probes and result output. Readiness uses prerequisite
  checks for startup.
- `interaction` supplies cancellable line input and terminal detection. Callers
  own prompt wording and acceptance rules. Exact-name removal approval belongs
  to `remove`; project-setup approval belongs to workspace creation.
- `state` owns versioned JSON snapshots, atomic writes, status storage, and
  advisory locking. `state.Paths` is the single owner of XDG path resolution and
  exposes `ConfigDir`, `StateDir`, `SSHDir`, and `DataDir`, plus per-instance and
  per-project module-data paths. Commands
  and workspace operations request the paths they need. Invalid state is reported
  to the caller.
- `provision` constructs private frozen archives and validates literal environment
  and APT inputs. Workspace creation supplies the resolved configuration and
  approved input locations.
- `process` owns subprocess streams, cancellation, process groups, and final Unix
  process replacement. Callers provide executable names and explicit argv.

## Configuration and invocation lifetime

| Concept | Owner and lifetime |
| --- | --- |
| Invocation | CLI constructs per execution; contains parsed general options, flags, and operands |
| Desired configuration | `config` merges the five ordered settings layers below for inspection or workspace creation |
| Saved configuration | `state` decodes an instance snapshot, including persistent mount bindings and mappings; readiness uses it for existing VMs |
| Module data | Module-owned contents in project-scoped host directories; retained across VM recreation |
| Dependencies | Entrypoint constructs services, streams, and asset access independently of a project |

User configuration lives under `${XDG_CONFIG_HOME:-$HOME/.config}/vbx/`. Project
configuration lives under `PROJECT/.vbx/`. Local relative paths resolve from the
user configuration directory; project-relative paths resolve from the project.

### Configuration precedence

All declarative settings use this order, from least to most important:

1. **VBX defaults.** Built-in configuration defaults.
2. **VBX module settings.** Settings from modules selected at the global level,
   including built-in module selections and modules named in user global settings.
3. **User global settings.** Explicit settings in the user configuration directory.
4. **Project module settings.** Settings from modules selected by the project.
5. **Project config settings.** Explicit settings in `PROJECT/.vbx/`.

Module settings merge in selection order within their layer; later modules
have higher priority. VBX reads module selections before merging their settings,
so settings from globally selected modules remain below explicit user settings.
Project-selected modules can override user global settings, and explicit project
settings can override those module settings. Selecting a module in both scopes
contributes its settings at both module layers; installation still runs once.

The order applies to resources, launch settings, mounts, modules, APT requirements,
environment-file declarations, and persistence. Field-specific merge rules remain
consistent across all five layers:

- Scalars use the highest-priority explicit value. An omitted value preserves
  the lower-priority value.
- Nested mappings merge by field.
- Mount entries merge by guest `target`; the higher-priority complete entry wins.
- Persistence entries merge by exact `guest`; the higher-priority complete entry
  wins, including its data path and declaring storage scope.
- Module, APT, and environment-file lists append in layer order and deduplicate
  identical entries. Module installation keeps first-selection order. Distinct
  APT constraints remain requirements for the combined solver phase.
- Environment assignments from the resulting environment-file sequence retain
  their existing last-assignment-wins behavior.

Module-level module selections append to that layer's selection sequence;
repeated selections contribute settings once per module layer and install once
overall. Embedded module settings require absolute or home-relative host paths
for mounts and environment files because embedded assets have no host-relative
directory.

Every source uses the same configuration schema for its settings. A sibling
`persistence.yaml` supplies only `persistence.paths`, with the same nesting as
`config.yaml`. Within a source, VBX reads `persistence.yaml` before `config.yaml`;
inline declarations therefore win over that source's sidecar declarations.
Duplicate keyed entries within one file remain errors.

Validation checks individual declarations and the final effective configuration.
Cross-source overlap checks operate on winning entries, so an overridden mapping
does not continue to reserve its former data path. Parent/child guest mappings
and conflicting effective data paths still fail validation. Resolution retains
each winning entry's origin and storage scope for capture and diagnostics.

Module `apt.txt` requirements join `apt.packages` at the declaring module's
layer, before that module's inline package declarations. The resolver validates
and deduplicates the combined requirements while preserving distinct constraints.
Bundle creation uses the resolved package list directly.

`show-project-effective-setup` prints the resulting effective configuration,
including combined `apt.packages` and merged `persistence.paths`, followed by a
home-overlay and persistence section. Workspace creation captures the same
effective configuration. Existing VMs and provisioning retries retain their
captured configuration until explicit removal and recreation.

`provision.WalkHome` owns shared metadata traversal and collision validation for
inspection and bundle capture. It visits global
then project entries without following child symlinks. Directory entries merge;
project files replace global files at the same relative path. File/directory
conflicts fail validation. Inspection displays winning file types, modes, relative
sources, overridden sources, literal symlink targets, managed-environment metadata,
and expected persistence connections. It never opens home file contents or reads
environment values, executes no scripts, and allocates no VM or retained state.
The handler validates before printing and labels the result as current desired
declarative inputs because arbitrary installers and project scripts can change
the final guest home.

Configuration access follows the operation:

| Operation | Required inputs |
| --- | --- |
| Help | Command metadata |
| `name` | Project path and canonical identity |
| `show-project-effective-setup` | Project identity, current desired configuration, and home-overlay metadata |
| `show-host-ssh-config-line` | Host SSH configuration directory |
| `check-host-prerequisites` | Host platform, tools, and virtualization access |
| `status` | Project identity and Lima inventory |
| `list-vms` | Lima inventory and captured mounts |
| `stop`, `remove` | Project identity and lifecycle metadata |
| Retained-data deletion | Project path or validated identifier, lifecycle lock, VM existence, and project-data path |
| New workspace | Desired configuration and approved setup inputs |
| Existing workspace | Saved configuration, captured data bindings and mappings, and pending captured setup inputs |

Launch overrides apply to the current invocation. Desired configuration and saved
snapshots retain their values. A ready workspace returns its configuration as
operation-owned data.

## Safety and process boundaries

- Setup runs during creation, with captured inputs available for ordinary retries.
  Ready VMs retain their captured setup until explicit removal and recreation.
- Inconsistent VM/state combinations produce errors. Recovery remains an explicit
  user decision. Confirmed removal can clear orphaned state.
- Successful readiness precedes every editor, shell, or guest-command launch.
- Locks and temporary staging are released before process replacement. `start.go`
  and `exec.go` check cancellation immediately before invoking the replacer.
  Cleanup completes explicitly before `syscall.Exec`, whose successful handoff
  replaces the calling process.
- SIGHUP, SIGINT, and SIGTERM drive cancellation and bounded child cleanup.
  Lifecycle locks remain held through child-operation cancellation.
- Temporary staging and retained setup archives containing credentials use private
  locations and restrictive modes. Staging cleanup removes disposable files;
  private instance state retains archives needed for retries.
- Credentials remain literal data. Diagnostics describe operations and errors.
  The executable embeds public installation assets only.
- Guest provisioning runs as `dev`; scripts explicitly use guest sudo when needed.
  Module installers and project setup are trusted code with access to configured
  mounts and supplied credentials.

## Dependency direction

```text
main ──> command
  ├────> cli ──> command
  └────> commands ──> command
               ├──> workspace ──> config / project / state / provision / module
               │             └──> Lima / host / interaction
               └──> lower services needed by each command

Lima ──> process / config / project / state
state ──> config / module
provision ──> module / state
```

The command contract forms the dependency boundary between CLI parsing and
command execution. Lower services depend on domain and infrastructure packages.
`main` assembles the command collection and passes it into the parser.

## Testing boundaries

- CLI tests use synthetic commands to exercise parsing, help, and dispatch.
- Command tests invoke actual handlers with injected services and check operation
  ordering, validation, output, failure behavior, and launch argv.
- Workspace tests cover readiness, snapshot reuse, frozen retries, locks,
  staging cleanup, persistent mount bindings, and retention through removal.
- Module-contract tests use synthetic modules to verify definition validation,
  script environments, stage ordering, and installation markers.
- Persistence-mapping tests verify manifest parsing, path containment, overlap
  checks, overlay-collision rejection before VM startup, interruption-safe
  linking, populated-directory conflicts, reapplication on retries, connection
  checks on ready reuse, and recovery diagnostics that preserve guest-local data.
- Persistence tests verify stable project/module paths, private directory
  permissions, captured bindings on reuse, active guest-mount verification, and
  failures for unavailable storage. Retained-data deletion tests require an absent
  VM and verify project-identifier confirmation and project-scoped deletion.
- Real-VM persistence checks verify 9p mount identity and permissions, VS Code
  installation into prelinked paths, extension installation/update/removal and
  file watching, Pi session resumption after recreation, and missing-source
  detection before Lima starts the guest. They also verify the tools preserve
  the mapping links during normal use.
- Configuration tests cover all five layers, module reselection, sidecar/inline
  priority, keyed mount and persistence overrides, effective overlap checks,
  configuration-owned storage, and merged configuration inspection. Setup-inspection
  tests cover winning and overridden home entries, metadata privacy, symlink
  targets, persistence targets, deterministic output, read-only behavior, and
  collision consistency with archive capture.
- Lower-level tests cover configuration, identity, snapshots, bundles, terminal
  handling, and process cancellation independently.
- Registry tests add a command through an isolated factory list and verify its
  inclusion in help and execution.
- Command behavior tests run the corresponding handler: removal tests exercise
  `remove.go`, and launch-policy tests exercise `start.go`.

See [development](../development.md) for contributor instructions and
[testing and validation](../testing/validation.md) for test commands and VM checks.
