# Provisioning and optional Pi tooling

## 1. Select tools before creating a VM

New VMs use the official Ubuntu 26.04 LTS minimal cloud image (Resolute Raccoon).
Initial guest setup runs these commands as root before installing tools:

```sh
apt-get update
apt-get upgrade -y --with-new-pkgs --no-remove
```

These perform the guest equivalent of `sudo apt update` followed by
`sudo apt upgrade`. Upgrades may install required new dependencies and cannot
remove packages. VBX runs this only during initial setup (or retries after
failure). It leaves host packages and existing VMs unchanged. If an upgrade
requires a reboot, stop and start the VM when convenient.

Edit `config.local/config.yaml` in the VBX checkout. Replace `modules: []` with:

```yaml
modules:
  - node
  - pi
```

The core works with an empty module list. Projects can also select modules in
`.vbx/config.yaml`. Lists append and deduplicate; `modules: []` does not disable
inherited modules. Recipes run in explicit list order, without automatically
selecting other recipes. Each recipe declares its own APT prerequisites.

| Selection | Installation |
| --- | --- |
| Core bootstrap | CA certificates and curl, plus the image's SSH/sudo/base utilities |
| `apt.packages` | User-selected Ubuntu packages and optional version constraints |
| `dev-tools` | Optional full tool bundle: compiler/Make, Git/GH, PostgreSQL client, Python/pipx, editors, archives, and diagnostics |
| `node` | Compatible Node/npm toolchain; bootstraps NodeSource Node 24 LTS when needed |
| `pi` | Git, Pi CLI, subagents, and prompt templates. Requires existing Node/npm. |
| `pi-web-fetch` | Optional extension, Python/pipx, browser libraries, Puppeteer/Chrome, and sandboxed rendering check. Requires Pi first. |

Pi does not install Python, Chrome, web-fetch, compilers, GH, PostgreSQL, or
editors. Select the relevant module or add packages under `apt.packages`.
The example configuration selects Git and `build-essential` there; existing local
configuration files are never overwritten.

For example, add the following to local or project configuration:

```yaml
apt:
  packages:
    - build-essential
    - gh
    - postgresql-client
    - 'jq (>= 1.7)'
```

VBX combines these entries with the selected recipes' `apt.txt` requirements and
lets `apt-get satisfy` resolve them together. Conflicts or unavailable versions
stop setup. APT cannot remove packages. Custom repository bootstrapping remains
explicit installer code where needed, such as the NodeSource fallback.

To include web extraction, explicitly select modules in order:

```yaml
modules:
  - node
  - pi
  - pi-web-fetch
```

Lifecycle approvals do not certify upstream package security. See
[security notes](recipes.md#security-notes) for browser dependency considerations.

Installers use APT for system packages and the guest user for npm/Pi packages.
They never install packages on the host. Browser installation checks headless
rendering with Chromium's sandbox enabled. On Ubuntu's AppArmor-enabled kernel,
it grants user namespaces specifically to Chrome in the guest user's Puppeteer
cache. It leaves the global user-namespace policy unchanged.

Creation requires internet access. APT and npm resolve package versions. Script
approvals cover exact reviewed versions; the web-fetch policy approves
Puppeteer 24.43.1. The `node` module accepts Node >=22.19 and npm >=11.19, and otherwise
bootstraps the Node 24 LTS toolchain. Ordinary reuse neither reinstalls nor
refreshes tools.

npm uses strict lifecycle-script enforcement during setup, including npm invoked
by `pi install` and project setup. Unreviewed scripts and script failures stop
provisioning and block launch. When selected, pi-web-fetch checks its actual
Puppeteer dependency, including sandboxed rendering. New script-bearing
versions may require a reviewed policy update and VM recreation. See
[recipe contract and npm policy](recipes.md).

## 2. Add API keys if needed

Run from the VBX checkout:

```sh
cp -n config.local.example/env.example config.local/.env
chmod 600 config.local/.env
```

Edit `.env`, then replace `envFiles: []` in `config.local/config.yaml` with:

```yaml
envFiles:
  - .env
```

Keep only the keys you need. Pi can also use its normal interactive login.
`config.local/` is ignored by Git. Never put keys in committed YAML or home examples.

Environment files use single-line `NAME=VALUE` entries. Blank lines, comments,
optional `export`, empty values, and surrounding single/double quotes work.
Quotes wrap literal text. There is no variable expansion, command execution,
escape processing, or multiline syntax. Use single quotes for values containing
backslashes. An unquoted whitespace-prefixed `#` begins an inline comment.

Local environment files load first, then project `envFiles` in list order.
Later definitions win. Relative local paths resolve from `config.local/` and
relative project paths resolve from the project root. Each selected file must
exist. `envFiles: []` adds no files and does not clear inherited entries.

VBX reserves `HOME`, `USER`, `LOGNAME`, `SHELL`, `PATH`, `BASH_ENV`, `ENV`, and
names beginning with `VBX_` or `XDG_`. Use home shell configuration for custom PATH
entries. VBX already adds `~/.local/bin`.

VBX transfers values over SSH and stores the guest environment in mode-600
`~/.config/vbx/environment.sh`. Its managed commands, login shells, and Ubuntu
interactive Bash terminals load that file. New VS Code remote sessions and
terminals use the same environment. Custom shell startup files can override it.
Restart an already-running VS Code remote server if it has an older environment.

A failed creation retains a private host setup archive for retries. VBX removes
that archive after successful setup. The guest retains the environment file.
Project setup and guest tools can read these credentials and use guest sudo.
Approve only trusted projects. User scripts remain responsible for keeping
secrets out of their own output.

## 3. Add personal and project home files

Place personal configuration under `config.local/home/`, for example:

```text
config.local/home/.pi/agent/AGENTS.md
config.local/home/.pi/agent/settings.json.example
config.local/home/.pi/agent/mcp.json
```

The copied examples contain reference-only Pi preferences and a built-in Perplexity MCP
server that reads `PERPLEXITY_API_KEY`. There is no MCP adapter to install.

Projects can provide `.vbx/home/` with the same layout. VBX merges local files
first, then project files. Dotfiles and modes survive. Copies use guest ownership,
keep the home directory's own mode, and retain unrelated files. A conflicting
file replaces the whole file, including JSON settings. Read-only files and
directories also merge while retaining the winning modes. File/directory type
conflicts stop setup rather than deleting a directory's unrelated contents.
Include all required Pi packages when overriding `settings.json`.

VBX owns `~/.config/vbx/environment.sh` and its provisioning markers under
`~/.local/state/vbx/provision/`. Do not supply those paths through overlays.

## 4. Add optional project setup

Create `.vbx/setup.sh` in the project:

```bash
#!/usr/bin/env bash
set -euo pipefail
# Runs as the guest user with PWD at the primary project mount.
# Install project dependencies here. Use sudo only for guest system changes.
```

VBX invokes it with Bash; an executable bit is optional. Make the script
idempotent because a failed script may run again on retry. Its captured script
body stays fixed during retries; mounted project files remain live.

Creation order:

1. Base guest update/upgrade, combined APT requirements, and selected recipes.
   Recipes run as the guest user and invoke guest sudo explicitly where needed.
2. Local home files.
3. Project home files.
4. Environment installation and loading.
5. Project setup from the primary project directory.
6. Requested editor, shell, or command after success.

## 5. Create and use the environment

From the project directory:

```sh
vbx --workspace-only
vbx --exec pi --version
vbx --exec pi list
vbx --exec pi mcp list
vbx --exec pi
```

Approve project-controlled configuration when prompted. Pi starts only when
explicitly requested. VS Code and terminal workflows work without Pi.

Existing VMs retain their original image and setup snapshot. To apply changed
modules, environment values, home files, or VM settings to a ready environment,
back up any guest-local work you want to keep, then run:

```sh
vbx --remove
vbx --workspace-only
```

Removal deletes guest-local data and keeps host project files.

## 6. Recover from setup failure

VBX blocks launch, records `setup-failed`, and keeps the incomplete VM for
inspection. Treat failed creation as disposable. The failure message prints the
commands to remove and recreate the affected project VM.

1. Read the setup error and correct the configuration or installer inputs.
2. Remove the incomplete VM and create it again:

```sh
vbx --remove
vbx --workspace-only
```

Removal keeps host project files. It does not delete the failed VM automatically,
so you can inspect it first if useful:

```sh
limactl shell "$(vbx --print-name)"
```

Edits do not update a captured snapshot. Ordinary retries reuse captured inputs
and skip successful steps, which can help after a transient network failure.
Use removal/recreation to apply corrected inputs. Ready VMs remain unchanged.

For a missing-Node error, add `node` before `pi` in the configuration layer that
selects Pi, then recreate. Lists append across layers, so adding Node only in a
later project layer will not move it before an inherited Pi entry.

### Operation lock remains busy after a command exits

The lock file's presence does not block VBX; a running process holding its kernel
lock does. If a command reports a busy instance, check for another active VBX
operation first. To inspect holders on the host (when `fuser` is installed):

```sh
name=$(vbx --print-name)
lock="${XDG_STATE_HOME:-$HOME/.local/state}/vbx/locks/$name.lock"
fuser -v "$lock"
```

The launcher closes its lock descriptor for Lima commands and releases the lock
on exit. If a leftover Lima process holds a lock and no VBX operation is running,
stop only the affected instance directly:

```sh
name=$(vbx --print-name)
limactl stop "$name"
```

This ends the instance's Lima processes. Retry the intended operation.
If it remains busy, inspect the lock holders (for example with `fuser -v` if
available) before stopping any process. Never delete the lock file while a holder
may still be alive: that creates a different inode and can bypass mutual exclusion.
