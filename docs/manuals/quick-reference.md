# Quick reference

Run commands from the project root, or add `--project DIR`. VBX does not search
for a Git root. See [project identity](commands.md#what-counts-as-a-project).

## Common tasks

| Task | Command or page |
| --- | --- |
| Open default target (VS Code by default) | `vbx` |
| Open editor / shell / background VM | `vbx start --vscode` / `--terminal` / `--workspace-only` |
| Run a guest command | `vbx exec npm test` |
| Run a shell pipeline in the guest | `vbx exec -- bash -c 'make && ./app'` |
| Inspect current VM / all VMs | `vbx status` / `vbx list-vms` |
| Inspect desired setup without starting a VM | `vbx show-project-effective-setup` |
| Stop VM (preserve guest-local files) | `vbx stop` |
| Remove VM (delete guest-local files, keep retained data) | `vbx remove` |
| Permanently delete retained data after removing VM | `vbx remove-project-data` |
| Diagnose host requirements | `vbx check-host-prerequisites` |
| Find VM name | `vbx name` |
| Configure SSH clients / forward a guest service | [SSH access](commands.md#ssh-access) |
| Access private Git repositories | [Git authentication](credentials.md#git-authentication) |
| Back up guest-local files before removal | [Backup procedure](persistence.md#back-up-guest-local-files) |
| Clean up a moved or deleted project | [Recovery](troubleshooting.md#moved-or-deleted-projects) |
| Apply changed setup | [Recreate the VM](configuration.md#reuse-and-configuration-changes) |
| Fix setup, lock, or state errors | [Troubleshooting](troubleshooting.md) |
| Get command help | `vbx --help` / `vbx exec --help` |
| Handle CLI and guest exit codes | [Exit status](commands.md#cli-edge-cases-and-exit-status) |

## Configuration keys

Edit global `${XDG_CONFIG_HOME:-$HOME/.config}/vbx/config.yaml` or project
`.vbx/config.yaml`. Both require `version: 1`.

| Need | Key | Details |
| --- | --- | --- |
| CPU, RAM, disk | `resources.cpus`, `.memory`, `.disk` | [Schema](configuration.md#supported-settings) |
| Default target / editor folder | `launch.default`, `.folder` | [Schema](configuration.md#supported-settings) |
| Additional host directories | `mounts[].source`, `.target`, `.readonly` | [Mounts](configuration.md#example-project-configuration) |
| Ubuntu packages | `apt.packages[]` | [APT requirements](recipes.md#apt-requirements) |
| Bundled tools | `modules[]` | [Catalogue](recipes.md#selection) |
| Credentials and environment | `envFiles[]` | [Environment](credentials.md) |
| Data retained across recreation | `persistence.paths[].guest`, `.data` | [Persistence](persistence.md) |

See [merge rules](configuration.md#files-and-precedence): lists append, and empty
lists preserve inherited entries. Ready VMs keep their saved setup.

## Files and paths

| Purpose | Path |
| --- | --- |
| Global configuration, environment files, overlays | `${XDG_CONFIG_HOME:-$HOME/.config}/vbx/` |
| Project configuration | `.vbx/config.yaml`, optional `.vbx/persistence.yaml` |
| Global / project home overlays | Global `home/` / project `.vbx/home/` |
| Project setup script | `.vbx/setup.sh` |
| Guest account / project mount | `/home/dev` / `/workspaces/<project-slug>` |
| Instance snapshots, locks, SSH links, retained data | [Managed files](configuration.md#files-vbx-manages) |

Global and project `persistence.yaml` sidecars contain only `persistence.paths`.
See [provisioning](provisioning.md) for overlays and scripts.
