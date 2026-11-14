# Provisioning design

## Decision: creation-only execution

VBX provisions a persistent VM before opening a development tool. Successful
reuse performs no installation or configuration convergence. Explicit
removal/recreation applies changed inputs.

Snapshots keep ordinary retries predictable. Completion markers skip successful
steps after transient failures. Corrected inputs require a new VM. Installers
must tolerate partial execution; VBX provides no transactional rollback.

## Decision: self-contained modules

Each bundled module owns `apt.txt`, `setup.sh`, local helpers, policy files,
and functional checks. It does not load installer code from another module or a
shared installer directory. Small utility copies keep ownership explicit.

Users select prerequisites in order: `node`, `pi`, and optional `pi-web-fetch`.
The core has a closed registry of bundled names. Project customization uses
`apt.packages`, home overlays, environment files, and `.vbx/setup.sh`.

This boundary excludes remote discovery, a module dependency graph, provider
selection, a task DSL, service orchestration, update hooks, and uninstall hooks.

## Decision: native package-manager resolution

APT resolves all distinct requirements from configuration and selected modules.
VBX validates syntax but does not choose versions or solve dependencies.
`--no-remove` prevents package resolution from deleting installed packages.
Modules own any external repository bootstrap needed by their installers.

npm resolves JavaScript dependencies and enforces native lifecycle approval.
Unreviewed scripts fail setup. Approvals name reviewed package versions and live
with the module that needs them. Tests exercise actual npm behavior with synthetic
local packages, including child processes and failing approved scripts.

Exact script approval is separate from dependency vulnerability analysis.
Extensions execute with guest-user permissions after installation.

## Decision: separate tools from launcher workflows

Node is independently useful for development. Pi adds its CLI, subagents, and
prompt templates. Web-fetch owns Python and browser requirements because those
requirements originate in that extension. The general development bundle stays
optional. Editor and terminal workflows do not require Pi.

## Reference material

These systems informed the boundaries; VBX does not implement their artifact or
lifecycle protocols.

- [Pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md)
- [Docker Sandbox kit authoring](https://docs.docker.com/ai/sandboxes/customize/author/)
- [Dev Container Features](https://containers.dev/implementors/features/)
- [cloud-init package installation](https://docs.cloud-init.io/en/latest/reference/modules.html#package-update-upgrade-install)
- [npm strict lifecycle approval](https://docs.npmjs.com/cli/v11/using-npm/config/#strict-allow-scripts)

See [recipe contract](../manuals/recipes.md) for the executable interface.
