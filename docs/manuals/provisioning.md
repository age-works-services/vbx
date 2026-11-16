# Provisioning

Use this page to customize a new guest. For installed tools, see the
[module catalogue](recipes.md) and [APT settings](configuration.md#supported-settings).
For API keys, see [environment and credentials](credentials.md).

## Trust and project approval

During creation, a project with a `.vbx/` directory requires interactive
approval. Review its configuration, modules, mounts, home files, and setup script
before answering yes. VBX prompts:

```text
Apply project .vbx mounts, modules, home files and setup (with access to guest credentials)? [y/N]
```

Modules, setup scripts, and installed extensions are trusted code with guest
sudo access. They can read supplied credentials, modify writable host mounts,
and use the network. npm lifecycle approvals govern installation scripts only.
They do not sandbox installed programs. Keep secrets out of scripts and output.

## Home overlays

Place personal configuration under `~/.config/vbx/home/`, for example:

```text
~/.config/vbx/home/.pi/agent/AGENTS.md
~/.config/vbx/home/.pi/agent/settings.json.example
~/.config/vbx/home/.pi/agent/mcp.json
```

The optional examples under `config.local.example/home/` contain reference-only
Pi preferences and a Perplexity MCP server that reads `PERPLEXITY_API_KEY`.
`AGENTS.md` above illustrates a file you can supply yourself.

Projects can provide `.vbx/home/` with the same layout. VBX merges local files
first, then project files. Dotfiles and modes survive. Copies use guest ownership,
keep the home directory's own mode, and retain unrelated files. A conflicting
file replaces the whole file, including JSON settings. Read-only files and
directories also merge while retaining the winning modes. File/directory type
conflicts stop setup rather than deleting a directory's unrelated contents.
Include all required Pi packages when overriding `settings.json`.

VBX owns `~/.config/vbx/environment.sh` and its provisioning markers under
`~/.local/state/vbx/provision/`. Do not supply those paths through overlays.


## Project setup

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


## Creation order

1. Base guest update/upgrade and combined APT installation.
2. Selected module scripts, in explicit selection order.
3. Global home overlay, then project home overlay.
4. Environment installation and loading.
5. Persistent directory connections.
6. Project setup, then the requested editor, shell, or command after success.

Modules run before environment installation. Put commands that need your supplied
API keys in project setup, rather than expecting module installers to receive them.

## Base packages

New guests use Ubuntu 26.04 LTS minimal. Initial root setup runs:

```sh
apt-get update
apt-get upgrade -y --with-new-pkgs --no-remove
```

Upgrades can add required dependencies and cannot remove packages. This affects
the guest only. If an upgrade requires a reboot, stop and start the VM when
convenient. Creation requires network access.

## Changes and recovery

Ready VMs keep their captured setup. Follow
[applying configuration changes](configuration.md#reuse-and-configuration-changes)
to recreate, or [troubleshooting](troubleshooting.md) to inspect failed setup.
