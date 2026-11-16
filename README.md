# VBX

VBX gives each project directory a persistent Ubuntu development VM on a local
Linux x86_64 host. Keep source files on the host and run toolchains, coding
agents, and commands in the guest. VS Code Remote SSH and terminals share the
same machine. Lima and QEMU run the VM with KVM acceleration.

## Quickstart

### 1. Check the host

You need Linux x86_64, hardware virtualization, read/write access to `/dev/kvm`,
latest stable Lima (`limactl`), QEMU (`qemu-system-x86_64`, `qemu-img`), and OpenSSH.
Building VBX also requires Git, Make, and Go 1.25 or later.

Allow 4 GiB guest RAM plus memory for the host. The default 30 GiB guest disk is
sparse and grows with use. First creation downloads Ubuntu 26.04 LTS minimal,
upgrades guest packages, and installs your selected tools. Allow several minutes
or longer depending on the network and tool selection. Creation needs host and
guest internet access.

See [host setup](docs/manuals/host-setup.md) for missing prerequisites. Run VBX
as your normal user, without sudo.

### 2. Install

```sh
git clone https://github.com/age-works-services/vbx.git
cd vbx
make install
export PATH="$HOME/.local/bin:$PATH"
vbx check-host-prerequisites
```

`make install` builds and installs `~/.local/bin/vbx`. Add that directory to your
shell's persistent PATH. The executable embeds its runtime assets and does not
need the checkout to run. See [updating and uninstalling](docs/manuals/installation.md).

### 3. Select tools (optional, before creation)

Skip this step to try the base Ubuntu VM. Select development tools explicitly:
use the [global configuration example](docs/manuals/configuration.md#example-global-configuration)
for Git and compiler tools, or [modules](docs/manuals/recipes.md) for Node, Pi,
and AWS CLI. Supply [credentials](docs/manuals/credentials.md) and
[home files or project setup](docs/manuals/provisioning.md) as needed.

Choose these before first creation. Ready VMs keep their creation-time setup;
applying later changes requires [backup and recreation](docs/manuals/configuration.md#reuse-and-configuration-changes).

### 4. Open a project

Run from the intended project root. VBX uses the exact selected directory and
does not search for a Git root. Subdirectories and separate worktrees select
separate VMs. See [project identity](docs/manuals/commands.md#what-counts-as-a-project).

No configuration is required for the base VM:

```sh
cd /path/to/project
vbx start --terminal
```

The first start creates the VM and streams setup output. If the project has a
`.vbx/` directory, VBX asks you to approve its setup. Later starts reuse the VM.
The guest account is `dev`, home `/home/dev`. No coding agent starts automatically.
Verify the shell inside the guest:

```sh
whoami
pwd
```

Expect `dev` and `/workspaces/<project-slug>`. Host project files appear at that
mount, so edits there affect your host files. Run `exit` to return to the host
shell. Closing the shell leaves the VM running.

To open VS Code from the host, install host VS Code with **Remote - SSH**
(`ms-vscode-remote.remote-ssh`). Ensure `code` is in PATH. Run:

```sh
vbx show-host-ssh-config-line
```

Add the printed `Include` line near the beginning of `~/.ssh/config`, then run:

```sh
vbx start --vscode
```

Bare `vbx` uses VS Code by default. Use `vbx start --workspace-only` for a
background VM or `vbx exec COMMAND [ARG...]` for a guest command.

### 5. Stop or recreate

From the host, stop the VM when you are done:

```sh
vbx stop
```

Stopping preserves guest-local files and installed tools. Ready VMs retain their
creation-time setup. To adopt configuration changes, [back up guest-local work](docs/manuals/persistence.md#back-up-guest-local-files),
then run:

```sh
vbx remove
vbx start --terminal
```

Removal requires confirmation and preserves host project files and explicitly
[retained data](docs/manuals/persistence.md). See
[troubleshooting](docs/manuals/troubleshooting.md) if creation fails.

## Isolation and trust

Guest commands run in a separate Linux VM. Writable host mounts (including the
project), supplied credentials, and unrestricted guest networking remain exposed
to guest programs. Setup scripts and installed tools have guest sudo access.
Use a dedicated worktree for unattended agent runs and provide only needed
mounts and credentials. Removing a VM does not undo host-file edits or revoke keys.
See [trust and approval](docs/manuals/provisioning.md#trust-and-project-approval).

## Documentation

- [Quick reference](docs/manuals/quick-reference.md)
- [Commands](docs/manuals/commands.md)
- [Troubleshooting](docs/manuals/troubleshooting.md)
- [All user manuals and contributor documentation](docs/README.md)

## License

[MIT](LICENSE). Copyright (c) 2026 VBX contributors.
