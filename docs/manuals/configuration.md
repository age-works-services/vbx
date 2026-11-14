# Configuration

## Guest account

New VMs use account `dev` with home `/home/dev`. Lima keeps its default numeric
UID mapping to the host user, so shared-file ownership does not depend on the
account names matching. VBX does not hard-code UID 1000.

VMs retain their creation-time account and home. VBX does not rename accounts
in existing guests.

## Files and precedence

1. Built-in defaults: 2 CPUs, 4GiB memory, 30GiB disk, VS Code launch.
2. Local defaults: `config.local/config.yaml` in the VBX checkout.
3. Project settings: `.vbx/config.yaml` in the project directory.

Configuration files are optional. Each file must include `version: 1`.
Project scalar settings override local settings. Mount lists append in that order;
repeated targets and overlapping host directories are errors. `mounts: []` adds no
mounts; it does not clear a previous layer. Module lists append and deduplicate.
Environment-file lists append, and later files override earlier variable values.
APT package lists also append and deduplicate identical requirements. Different
version constraints remain separate so APT can satisfy them together or fail.
Empty lists do not clear inherited entries. Recipes run in the resulting module
list order, without implicit module dependencies.

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
from the project root. Local relative sources resolve from `config.local/`.
Absolute paths and `~/` are supported. Environment-variable expansion is not performed.

## Supported settings

| Setting | Values |
| --- | --- |
| `version` | `1` |
| `resources.cpus` | Positive integer |
| `resources.memory` | Positive integer followed by `MiB`, `GiB`, or `TiB` |
| `resources.disk` | Positive integer followed by `MiB`, `GiB`, or `TiB` |
| `launch.default` | `vscode`, `terminal`, or `workspace-only` |
| `launch.folder` | Absolute guest path opened by VS Code |
| `mounts[].source` | Existing host directory |
| `mounts[].target` | Immediate child of `/workspaces`, such as `/workspaces/api` |
| `mounts[].readonly` | `true` or `false`; defaults to `false` |
| `modules[]` | Explicitly ordered bundled recipes: `dev-tools`, `node`, `pi`, or `pi-web-fetch` (select after `pi`) |
| `apt.packages[]` | Package name, optionally `package (>= VERSION)`, `package (<= VERSION)`, or `package (= VERSION)` |
| `envFiles[]` | Existing dotenv-style file; paths follow the same local/project rules as mount sources |

The primary project is writable at `/workspaces/<project-slug>`. Terminals and
explicit commands start there. VS Code opens that directory when no extra mounts
exist, or `/workspaces` when additional directories are mounted. `launch.folder`
overrides the editor folder only.

Use two-space indentation and the block structure shown above. The parser
supports plain scalars, single-quoted strings (escape `'` as `''`), JSON-compatible
double-quoted strings, comments, and empty lists (`mounts: []`, `modules: []`,
`envFiles: []`, or nested `apt:` / `packages: []`). It rejects other YAML constructs,
unknown settings, duplicate keys, and tabs. Configuration is data; it is never
sourced as shell code. No external YAML parser is required.

## Validate before creation

From the project directory:

```sh
vbx --render
```

This validates the desired YAML configuration and referenced file/directory paths,
then prints the Lima template without creating a VM. It does not display modules
or credentials, or parse environment-file contents. Lima validates the template
again during creation; VBX checks environment-file contents before starting a VM.

Creating a VM with a project `.vbx/` directory requires interactive approval.
APT requirements allow package names (optionally `:architecture`) and a single
`>=`, `<=`, or `=` Debian version constraint. Alternatives, flags, URLs, shell
expressions, and arbitrary APT directives are unsupported. Names and constraints
are data; APT resolves them against configured guest repositories. VBX prevents
package removals and does not implement its own version solver.
See [provisioning](provisioning.md) for module selection, optional API-key files,
home overlays, project setup, and failure recovery.

## Reuse and configuration changes

Ready VMs reuse their saved configuration. Editing either configuration file,
home overlays, environment-file contents, modules, or setup scripts does not
change a ready VM.
To apply changes to a ready VM:

```sh
vbx --remove
vbx --vscode
```

Removal discards guest-local data. Failed creation leaves a disposable incomplete
VM for inspection. Correct its setup inputs, then remove and recreate it. Ordinary
retries use the original snapshot; there is no in-place setup refresh.

VBX manages its own `vbx-` instances and does not adopt other VMs.

## Files VBX manages

| Files | Location |
| --- | --- |
| Lima VM data | Lima's configured home (`LIMA_HOME`, normally `~/.lima`) |
| Saved configuration, creation status, and retained setup archive | `${XDG_STATE_HOME:-$HOME/.local/state}/vbx/instances/` |
| Operation locks | `${XDG_STATE_HOME:-$HOME/.local/state}/vbx/locks/` |
| SSH configuration links | `${XDG_CONFIG_HOME:-$HOME/.config}/vbx/ssh/` |

VBX prints the SSH `Include` line with `vbx --ssh-config`. It does not edit
`~/.ssh/config`. Creation and startup are locked per project; the lock is released
before opening the editor, shell, or command so multiple sessions can coexist.
