# Go migration and command-line redesign

## Goal

Move VBX's host application to Go and retain Bash for guest provisioning. Replace
mutually exclusive action flags with explicit commands and scoped options.

Adding an ordinary command should require one implementation file under
`internal/commands/`, plus its tests and documentation. It should not require
editing a central command switch, registration list, or help table.

This document is an implementation plan. The current Bash/Perl CLI remains
unchanged until the migration ships. The CLI and configuration-location decisions
below describe the proposed release behavior.

## 1. Scope and constraints

- Go owns host argument parsing, configuration, validation, project identity,
  state, locking, subprocesses, VM lifecycle, and provisioning bundle creation.
- Bash owns guest provisioning, module entrypoints, and project setup scripts.
  No Go runtime or VBX Go executable is required inside the guest.
- Keep module-local assets and specialized checks, including the existing browser
  JavaScript check. This migration does not rewrite tools installed by modules.
- Ship a single host executable containing the built-in template, guest scripts,
  and bundled modules. Users still install Lima, QEMU, SSH, and optional editors.
- Preserve creation-only provisioning, frozen inputs, explicit project approval,
  guest-user execution, explicit guest sudo, and removal confirmation.
- Keep Linux x86_64 as the supported VM host for this migration. Portable Go code
  and macOS build checks prepare for a separate macOS backend/host-support change.
  A successful macOS build does not establish VM support there.
- Do not add refresh, update, uninstall, remote module discovery, or automatic
  module dependency resolution as part of this work.

## 2. Command-line contract

### Commands

| Command | Behavior | Replaces |
| --- | --- | --- |
| `start` | Create/start/reuse the VM and launch the configured target unless overridden | Startup portion of launch actions |
| `stop` | Stop an existing VM | `--stop` |
| `remove` | Confirm removal, then delete the VM and its VBX state | `--remove` |
| `exec COMMAND [ARG...]` | Ensure the VM is ready and run a guest command | `--exec COMMAND...` |
| `render` | Validate current desired configuration and print Lima YAML | `--render` |
| `name` | Print the canonical project VM name | `--print-name` |
| `ssh-config` | Print the host SSH Include line | `--ssh-config` |
| `check` | Check host prerequisites | `--check` |

### General options

| Option | Meaning |
| --- | --- |
| `--project DIR` | Select a project; default to the current working directory |
| `-h`, `--help` | Print root or selected-command help without running the command |

General options are syntactically valid on either side of the command. Commands
only load the inputs they need. For example, `check` and `ssh-config` do not
validate the selected project directory or parse project configuration.

### Command options

| Command | Option | Meaning |
| --- | --- | --- |
| `start` | `--vscode` | Open VS Code after readiness |
| `start` | `--terminal` | Open a guest login shell after readiness |
| `start` | `--workspace-only` | Start without opening a tool, overriding the configured target |

`--vscode`, `--terminal`, and `--workspace-only` are mutually exclusive.
Command-specific options must follow their command. Other commands initially
have no command-specific options.

### Defaults and intentional changes

- `vbx start` ensures readiness and launches the configured target. Use the
  captured `launch.default` for an existing VM, or the newly resolved default on
  first creation. A configured `workspace-only` target opens no tool.
- Bare `vbx` defaults to `vbx start` with identical behavior. General-options-only
  invocations, such as `vbx --project DIR`, also default to `start`.
- `vbx start --vscode`, `vbx start --terminal`, and
  `vbx start --workspace-only` override the configured target for that invocation
  without changing configuration or the saved snapshot.
- Keep `launch.default` and `launch.folder` in configuration. The default applies
  equally to explicit and defaulted `start` invocations.
- Treat the command syntax as an intentional CLI break. Reject old action flags
  with a replacement hint, and publish the migration table above. Do not retain
  two full parsers or a permanent Bash compatibility launcher.
- Keep `--project`, `-h`, and `--help`. Move the old `--workspace-only` action to
  the `start --workspace-only` command option.

### Supported ordering

Both forms must work:

```text
vbx [GENERAL OPTIONS] COMMAND [COMMAND OPTIONS]
vbx COMMAND [GENERAL OPTIONS AND COMMAND OPTIONS]
```

General options may also appear on both sides, provided an option occurs once:

```sh
vbx --project ~/work/app start --vscode
vbx start --project ~/work/app --vscode
vbx start --vscode --project ~/work/app
vbx --project ~/work/app start --help
vbx --project ~/work/app remove
vbx remove --project ~/work/app
```

Additional rules:

- Accept `--project=DIR` as well as a space-separated value.
- Resolve relative option paths against the invocation working directory.
- Reject repeated value options, including repetitions across the command
  boundary. Reject repeated command options rather than silently choosing one.
- Reject unknown commands, unknown options, missing values, and surplus positional
  arguments. `vbx start stop` is invalid.
- `vbx --vscode start` is invalid because `--vscode` belongs to `start`.
- `vbx --help` shows root help. `vbx start --help` and `vbx --help start` show
  `start` help. Help requires no Lima, configuration reads, locks, or approval.
- Do not search arbitrary argument values for command names. For example,
  `vbx --project start stop` selects project directory `start` and command `stop`.

### Guest arguments and `exec`

VBX option parsing stops at the first guest command operand, or at an explicit
`--` after `exec`. Every later argument belongs to the guest, including strings
that resemble VBX options or commands.

```sh
vbx --project ~/work/app exec npm test
vbx exec --project ~/work/app npm test
vbx exec --project ~/work/app -- npm test -- --watch
vbx exec printf '%s\n' '--project' 'a value with spaces'
vbx exec -- bash -c 'make && ./app'
```

In `vbx exec npm --project other`, `--project other` belongs to npm. It does not
change VBX's selected project. Use `--` when the guest executable itself starts
with a dash. Reject an empty guest command, including `vbx exec --`.

Store guest arguments as a string slice and pass them unchanged. Never join them
into host shell code. Preserve the existing guest environment wrapper by passing
arguments after the fixed Bash script and using `exec "$@"` inside that script.
`vbx exec --help` shows VBX exec help; `vbx exec npm --help` invokes npm help.

## 3. Go layout and command registration

Proposed layout:

```text
cmd/vbx/main.go                  # Build dependencies, run CLI, choose exit code
assets.go                       # Root embed package for checked-in assets
internal/
  cli/                          # Two-stage parser, help, command contracts
  commands/
    registry.go                 # Generic factory registration and validation
    start.go
    stop.go
    remove.go
    exec.go
    render.go
    name.go
    ssh_config.go
    check.go
    *_test.go
  app/                          # Shared lifecycle workflows and dependencies
  config/                       # Configuration parsing, merging, validation
  project/                      # Canonical paths, stable identity, workspace path
  state/                        # Snapshots, status, locks, legacy snapshot reader
  backend/lima/                 # Lima operations and output decoding
  process/                      # argv execution, streams, signals, exit handling
  provision/                    # APT declarations, dotenv, tar bundle creation
lima/default.yaml               # Embedded built-in template
modules/                       # Embedded bundled modules, existing ownership
scripts/                       # Guest-only scripts after migration
```

### Registration pattern

Use a small factory registry inside the `commands` package. Each command file
registers its own factory in `init()`:

```go
// internal/commands/status.go (illustrative future command)
func init() {
    register(newStatusCommand)
}
```

- A factory receives application dependencies and returns a fresh command
  descriptor. The descriptor owns its name, summary, usage, flag bindings,
  argument validation, and execution handler.
- `commands.New(app)` instantiates all registered factories, validates unique
  names and flag scopes, sorts commands for stable help, and returns the command
  set. It contains no command-name-specific switch.
- `main.go` imports `internal/commands` once. Go compiles all ordinary `.go` files
  in that package, so adding `commands/xyz.go` adds its registration automatically.
- Initialization only registers factories. It performs no I/O and captures no
  mutable invocation state. Each invocation gets fresh options and flag sets.
- Generate root help from descriptors and command help from descriptor metadata
  and flag definitions. Keep legacy replacement hints as temporary migration
  metadata, separate from normal command dispatch.
- Validate collisions between command flags and general flags. Fail tests for
  duplicate command names, missing help, or invalid descriptors.
- Keep `cli` independent of `commands`, and keep application services independent
  of both. This avoids import cycles and makes handlers directly testable.
- An invocation without a command resolves the registered `start` command.
  Defaulted and explicit invocations use the same handler, options, and launch
  target resolution. No separate implicit-invocation mode is needed.

A new command that uses existing services needs only its command file, tests, and
user documentation. A new backend capability can also require a service method.
There is no runtime plugin loading or filesystem scanning for command source.

### Parsing implementation

Use Go's standard `flag` package with a small two-stage dispatcher. Its
stop-at-first-positional behavior matches this CLI when applied deliberately:

1. Parse general options until the first positional token, the command name.
2. Resolve that name through the registry. If absent, select `start`.
3. Construct the command flag set with both general and command-specific flags.
   Bind general flags to the same invocation options as stage one.
4. Parse command-side flags until the first positional token or `--`.
5. For `exec`, preserve all remaining tokens as guest argv. For other commands,
   apply the descriptor's positional-argument validation.
6. Handle help and validate options before creating any external side effects.
7. Invoke the handler with a context and injected services.

Track option occurrences across both stages so duplicates cannot slip through.
Do not rely on `flag` alone to implement scope checks, duplicate rejection, or
mutual exclusion. Use `ContinueOnError` and return errors to the caller rather
than exiting from a parser or command handler.

## 4. Host implementation responsibilities

### Migration map

| Current implementation | Go destination |
| --- | --- |
| `vbx` help, parser, action dispatch | `cli`, `commands` |
| `vbx` startup, approval, SSH registration | `app`, `backend/lima` |
| `scripts/host.sh` | `project`, host checks in application services |
| `scripts/config.pl` | `config` |
| `scripts/apt-deps.pl` | `provision` APT validation |
| `scripts/environment.pl` | `provision` dotenv parsing and literal shell export rendering |
| `scripts/provision-host.sh` | `provision`, `app` |
| `scripts/provision-guest.sh` | Retain as guest Bash |
| `scripts/apt-install.sh`, `guest-shell-system.sh`, `npm-policy.sh` | Retain as guest Bash |
| `modules/*/setup.sh` and local helpers/assets | Retain module ownership |

### Shared workflows

- Provide a shared `EnsureReady` workflow for `start` and `exec`. Preserve
  approval, snapshot capture, startup, retry, provisioning, and SSH registration
  order. Only `start` launches the configured target; `exec` runs its guest argv.
- Dispatch `stop` and `remove` without `EnsureReady`. Neither may create a VM,
  start one for inspection, parse changed project configuration, or run setup.
- `name` only needs project identity. `render` reads desired configuration and
  renders locally without Lima. `ssh-config` only needs the SSH configuration
  location. Full host preflight belongs to VM startup and `check`.
- Model lock ownership with a scoped resource and guaranteed cleanup. Keep the
  existing lock path and compatible OS advisory locking during transition.
  Ensure child processes cannot inherit the lock descriptor.
- Release lifecycle locks before starting an editor, interactive shell, or guest
  command. Preserve multiple concurrent guest sessions.
- Keep removal's TTY requirement and exact VM-name confirmation. Stop a running
  VM before deleting it. Preserve VBX state if stop or deletion fails.

### Configuration and project identity

- Preserve current configuration precedence, list append/deduplication rules,
  path bases, closed module names, defaults, and validation restrictions.
- Port the restricted version-1 configuration grammar to Go first. Use golden
  fixtures to preserve accepted and rejected syntax. A general YAML parser or
  expanded YAML grammar can follow in a separate change.
- Preserve dotenv literal handling, reserved variable rejection, and secret-free
  diagnostics. Configuration and environment files remain data.
- Preserve APT normalization and constraints. Let guest APT resolve versions.
- Match canonical project paths, ASCII slug normalization/truncation, SHA-256
  input bytes, 12-character digest suffix, and workspace names exactly. A rename
  would otherwise select a different VM for an existing project.
- Replace host GNU utility calls with Go filesystem, hashing, archive, and string
  operations. Keep the external commands that provide actual VM/editor services.

### State compatibility

- Preserve XDG instance, lock, and SSH-link locations and existing Lima names.
- Read existing NUL-delimited `config` snapshots through a strict legacy decoder.
  Write new snapshots as versioned JSON, with an explicit format version.
- Detect legacy versus new snapshots unambiguously and reject unknown versions.
- Keep recovery simple and user-controlled. If inconsistent state prevents normal
  startup, report the problem and stop. Do not automatically repair, delete, or
  replace a VM or its saved state. This includes a missing VM with leftover
  instance state. The user decides whether to inspect, fix, or remove it.
- Normal creation with no existing VM or instance state remains automatic.
  Ordinary provisioning retries with valid captured inputs remain supported.
- Leave legacy snapshots unchanged during normal reuse. Document that the old
  Bash launcher cannot read newly written Go snapshots; switching back is not a
  supported rollback for newly created instances.
- Preserve `creating`, `setup`, `setup-failed`, and `ready` state transitions,
  retained `provision.tar`, and guest completion markers.
- Existing failed setups retry the captured archive, even after the binary or
  desired configuration changes. Existing ready VMs do not rerun provisioning.
- Use atomic writes for new state and restrictive permissions. Keep state
  directories private and credential-bearing archives/environment files at 0600.

### Subprocesses and errors

- Centralize subprocess execution through an injectable runner. Use explicit
  argument slices, stream connections, and controlled command logging.
- For final terminal, guest-command, and editor handoff, clean temporary files
  and release locks before Unix process replacement. Preserve the inherited
  streams and terminal so Lima/editor exit status and signals pass through.
- During lifecycle operations, catch SIGINT/SIGTERM and perform cleanup before
  exiting. Define graceful child cancellation and a bounded wait, with forced
  termination only as a fallback. Do not rely on Go's default immediate-kill
  cancellation. Avoid forwarding terminal signals twice to the same child.
- Test lifecycle cancellation separately from final handoff, including temporary
  credential cleanup, lock release, and child-process behavior.
- Keep machine-readable results on stdout and diagnostics on stderr.
- Use exit code 2 for CLI usage errors and 1 for operational failures. Preserve
  guest command exit codes through process replacement. Use 130 for host SIGINT
  and 143 for host SIGTERM after cleanup.
- Redact secrets by construction. Do not log environment-file contents, generated
  exports, or provisioning archive payloads.

## 5. Embedded assets and local configuration

- Embed only the built-in template, guest scripts, and known module directories
  with `go:embed`. Keep source files in their existing ownership locations.
- Never embed `config.local/`, project `.vbx/`, private home overlays, environment
  files, development state, or arbitrary checkout contents.
- Account for embedded filesystem permission normalization and dotfile exclusions.
  Use explicit inclusion rules and asset-mode metadata where executable assets
  need modes. Check that every selected module's complete required contents ship.
- Build private tar bundles with Go's archive APIs. Preserve home-overlay file
  modes, symlinks, and precedence. Match current setup-script symlink handling.
  Reject unsupported special files and unsafe archive entries with clear errors.
- Keep guest bundle layout and transport compatible during migration: stream the
  tar archive to a fixed guest Bash wrapper, extract privately, run provisioning,
  then clean up. Never stage credentials in mounted project directories.

Use `~/.config/vbx/` for local configuration, honoring `XDG_CONFIG_HOME` when set:

```text
~/.config/vbx/
  config.yaml
  home/
  ssh/          # VBX-managed SSH links, separate from user configuration
```

- Resolve the directory as `${XDG_CONFIG_HOME:-$HOME/.config}/vbx` and require an
  absolute XDG path. Keep project settings at `PROJECT/.vbx/`.
- There is one current user. Move the existing checkout's `config.local/` contents
  manually during cutover and review relative paths. Do not add `--config-dir`,
  legacy-directory discovery, automatic copying, or a configuration migration tool.
- Resolve local relative mount/env paths against this directory. Project-relative
  paths continue to resolve against the project root.
- Local `config.yaml` and `home/` remain optional. Their absence uses built-in
  defaults plus any project configuration.
- Never overwrite private files during installation. Keep state and lock paths
  unchanged, separate from this configuration directory.
- Existing VM reuse continues to use its captured snapshot. The new directory
  location affects desired configuration rendering and future VM creation.

## 6. Implementation sequence

### Phase 1: Capture the behavioral baseline

- Record current CLI, identity, configuration, environment, APT, and bundle
  behavior as fixtures. Use synthetic secrets only.
- Run the existing six shell suites and record skips and prerequisites from
  `docs/testing/validation.md`.
- Add cases for ready reuse, failed-setup retry, missing snapshots, destructive
  failures, and lock inheritance before replacing implementation boundaries.

**Exit criterion:** fixtures describe current behavior and explicitly identify the
intentional CLI/configuration-location changes in this plan.

### Phase 2: Introduce Go CLI structure

- Add `go.mod`, `cmd/vbx`, the command contracts, registry, and parser.
- Build a separate development executable at `bin/vbx`; leave the old root
  launcher intact during implementation.
- Implement generated help and every command's option/argument validation.
- Use fake application services while implementing parser and registration tests.
- Verify that adding a test-only command descriptor requires no dispatcher edit.

**Exit criterion:** all ordering, scoping, help, and guest argv tests pass without
requiring Lima or touching VM state.

### Phase 3: Port pure host logic

- Port identity, configuration parsing/merging, APT requirements, dotenv parsing,
  template rendering, and snapshot decoding to Go.
- Implement the fixed XDG local configuration location and asset embedding.
- Compare Go results against the existing fixtures. Test literal shell syntax,
  quoted paths, CRLF handling, invalid input, and secret-free errors.
- Implement `name`, `render`, and `ssh-config` using these components.

**Exit criterion:** offline commands and pure-data behavior match the baseline,
with documented intentional differences only.

### Phase 4: Port lifecycle and guest transport

- Implement process execution, Lima operations, host checks, locks, and state.
- Implement `check`, `stop`, and `remove`, including failure preservation.
- Port bundle generation, overlays, environment exports, and frozen-input retry.
- Implement `EnsureReady`, `start`, default-command resolution, and `exec`.
- Keep guest Bash and module execution order unchanged while validating Go
  transport. Avoid simultaneous guest refactoring that obscures regressions.

**Exit criterion:** mocked lifecycle tests pass for creation, reuse, retry,
concurrency, interactive approval, cleanup, command forwarding, and failures.

### Phase 5: Migrate tests and validate real execution

- Move host tests that source `vbx` or `host.sh` into Go unit/integration tests.
- Replace Perl invocations in provisioning and recipe tests with Go-level tests
  or bundles produced by the Go test harness. Do not add public CLI commands just
  to expose internal test helpers.
- Retain guest/module shell suites and actual npm policy fixtures. Keep APT and
  sudo test doubles so tests do not modify the development host.
- Run a disposable Linux VM walkthrough: creation, terminal, editor, exec,
  stop/start, removal, frozen retry, mounts, environment, and representative modules.
- Test Go adoption of an existing ready Bash-created VM and a failed VM with a
  retained archive. Confirm identity and guest completion markers remain stable.

**Exit criterion:** automated coverage and real-VM evidence meet the acceptance
checklist below, with skips reported explicitly.

### Phase 6: Cut over and publish

- Build release executables with a pinned Go toolchain and dependency checksums.
  Prefer `CGO_ENABLED=0` for the supported Linux release where dependencies allow.
- Keep Go build/vet/test jobs and build checks for intended future platforms.
  Clearly distinguish supported release targets from compile-only targets.
- Update installation, command, configuration, module-development, recovery, and
  testing documentation. Update examples and failure hints to the new commands.
- Document old flag replacements, bare invocation behavior, configuration location,
  user-controlled recovery, and state-format rollback limits.
- Manually move the current user's local configuration to `~/.config/vbx/`
  (or its XDG equivalent) and review relative paths before creating new VMs.
- Remove the Bash host launcher, host-only shell helpers, and Perl parsers once
  all callers and tests use Go. Keep guest scripts and module assets.
- Ensure installation never overwrites private local configuration.

**Exit criterion:** a user can install one executable and operate VBX without
host Perl, a host Bash launcher, or a source checkout.

## 7. Acceptance checklist

### CLI and extensibility

- Both `vbx --project DIR start --vscode` and
  `vbx start --vscode --project DIR` produce equivalent invocations.
- General options work on either side for every command, including `exec` before
  the guest argv boundary. Mixed placement works without lost values.
- Duplicate options across stages, unknown flags, incompatible launch options,
  missing values, invalid command placement, and surplus operands fail cleanly.
- Parser tests cover values that resemble command names, `--name=value` syntax,
  paths with spaces, explicit `--`, and guest flags that resemble VBX options.
- Help works without a usable project or VM backend and never starts a VM.
- `vbx` and `vbx start` both honor the configured launch target, using the saved
  default on reuse and the resolved default on creation. Launch options override
  it for one invocation only, including `start --workspace-only` to open no tool.
- A new `internal/commands/xyz.go` can register a command and appear in help
  without edits to `main.go`, the parser, or a central command list.

### Lifecycle, data, and security

- Existing project paths resolve to exactly the same VM and primary mount names.
- Existing ready VMs reuse their snapshot despite invalid desired configuration.
- Failed provisioning retries frozen inputs and releases the operation lock.
- Concurrent lifecycle operations fail predictably; guest sessions can coexist.
- Stop/remove never enter startup or provisioning. Failed removal preserves state.
- Inconsistent VM/state combinations produce a clear error without automatic
  repair or replacement. The user controls recovery.
- Bundles preserve required files, modes, symlinks, module order, and overlay order.
- Environment values remain literal and absent from diagnostics.
- Guest commands preserve argument boundaries, terminal behavior, signals, and
  exit status. Host shell evaluation never occurs for user-supplied argv.
- Private configuration does not appear in embedded assets or release artifacts.
- No supported host operation depends on the removed Perl or host Bash code.

### Validation commands

During migration, run the existing suites while their implementations remain:

```sh
bash test/test-host.sh
bash test/test-vbx.sh
bash test/test-provision.sh
bash test/test-recipes.sh
bash test/test-pi-modules.sh
bash test/test-module-boundaries.sh
```

As Go replaces host coverage, run:

```sh
go test ./...
go test -race ./...
go vet ./...
go build -o bin/vbx ./cmd/vbx
```

Run retained guest/module suites alongside Go tests. Use the prerequisites and
real-VM procedures in `docs/testing/validation.md`, updating commands as the CLI
cuts over. A cross-build or mocked test alone does not prove VM compatibility.
