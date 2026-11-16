# Developing bundled modules

For user selection and package policy, see the [module catalogue](manuals/recipes.md).

## File contract

```text
modules/<name>/
  config.yaml       # Optional module settings
  apt.txt           # Optional OS package requirements
  setup.sh          # Optional, blocking installation entrypoint
  persistence.yaml  # Optional settings under persistence.paths
  ...               # Local helpers, policies, assets, and checks
```

Each module provides settings, `setup.sh`, or both for its single purpose.
Settings use the [five-layer configuration merge](manuals/configuration.md#files-and-precedence).
Embedded module settings use absolute or `~/` host paths for mounts and environment
files; they have no host directory for resolving relative paths.
See [persistence declarations](#persistence-declarations) for the module format and
[project persistence](manuals/persistence.md) for retention behavior.

All installer code stays in its owning module directory. Modules do not load
installer code from the framework or sibling modules. The core provides generic
transport, APT resolution, ordering, markers, and setup-wide npm enforcement.

The closed registry in `internal/config/merge.go` accepts bundled names only. Adding a
bundled module requires registering its identifier there.
Use `.vbx/setup.sh` for project-specific commands.

The [AWS CLI module](../modules/aws-cli/README.md) downloads its installer from
AWS directly and keeps its tests with its implementation.


## APT requirements

`apt.txt` contains one requirement per line. Blank lines and full-line comments
work. Requirements use package names (optionally `:architecture`) and an optional
single `>=`, `<=`, or `=` Debian version comparison:

```text
# Packages from configured guest repositories
curl
git
jq (>= 1.7)
```

The settings resolver combines these with `apt.packages` at the module's layer,
reading `apt.txt` before that module's `config.yaml` package declarations. It
deduplicates identical requirements and retains distinct constraints.
`show-project-effective-setup` includes the combined list, and provisioning captures it
without adding further module requirements. APT resolves them with
`apt-get satisfy --no-remove --no-install-recommends`.
Alternatives, URLs, flags, and arbitrary APT directives
are unsupported. Conflicts and unavailable versions stop setup.

A module owns external repository bootstrap in its setup script. The `node`
module uses the signed NodeSource Node 24 repository if compatible Node/npm are
absent. Do not also request incompatible Ubuntu Node/npm packages in `apt`.

Node itself is system-installed through APT. npm global packages use
`~/.local/lib/node_modules`, with commands in `~/.local/bin`.

## Persistence declarations

A persistence-only module contains `persistence.yaml`:

```yaml
persistence:
  paths:
    - guest: ~/.tool/sessions
      data: sessions
```

`guest` is a directory beneath the guest home, and `data` is a relative directory
beneath the module data mount. Paths use literal components; `.` and `..`, empty
components, control characters, variable expansion, and overlapping mappings are
rejected after precedence resolution. A module provides settings, `setup.sh`, or
both for its single purpose. The core owns validation, storage, and linking.

Installation scripts receive `VBX_MODULE_DIR` for captured module files and
`VBX_MODULE_DATA_DIR` for their project-scoped data mount. Captured module files
are removed after successful provisioning; copy required runtime assets to their
installed locations.

## Execution and failure

VBX captures complete selected module directories, requirements, environment,
home overlays, and project setup before creation. The combined APT phase runs
before modules. Each `setup.sh` runs as `dev` from the primary project mount and
uses `sudo -n` explicitly for privileged guest operations.

Scripts should use `set -euo pipefail`, tolerate partial execution, locate local
files through their own script path, and check functionality before success.
Completed-step markers support ordinary retries of captured inputs.

Failure blocks launch and keeps the disposable VM for inspection. Correct the
inputs, preserve valuable guest-local data, then use `vbx remove` and
`vbx start --workspace-only`.
Ready VMs retain their original setup.

