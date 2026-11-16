# Provisioning design

## Creation-only execution

VBX provisions a persistent VM before opening a development tool. Successful
reuse performs no installation or configuration convergence. Explicit
removal/recreation applies changed inputs.

Snapshots keep ordinary retries predictable. Completion markers skip successful
steps after transient failures. Corrected inputs require a new VM. Installers
must tolerate partial execution.

## Self-contained modules

Each bundled module owns its APT requirements, installation script or persistence
declarations, and any local helpers, policies, and functional checks. It does not load installer code from another module or a
shared installer directory.

Users select prerequisites in order: `node`, `pi`, and optional `pi-web-fetch`.
The core has a closed registry of bundled names. Project customization uses
`apt.packages`, home overlays, environment files, and `.vbx/setup.sh`.

## Native package-manager resolution

APT resolves all distinct requirements from configuration and selected modules.
VBX validates requirement syntax and passes the constraints to APT.
`--no-remove` prevents package resolution from deleting installed packages.
Modules own any external repository bootstrap needed by their installers.

npm resolves JavaScript dependencies and enforces native lifecycle approval.
Unreviewed scripts fail setup. Approvals name reviewed package versions and live
with the module that needs them. Extensions execute with guest-user permissions
after installation.

## Tool selection

Node is independently useful for development. Pi adds its CLI, subagents, and
prompt templates. Web-fetch owns Python and browser requirements because those
requirements originate in that extension. Users select the general development
bundle and Pi explicitly.

See [recipe contract](../manuals/recipes.md) for the executable interface.
