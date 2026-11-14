# VBX

VBX gives each project directory a persistent Linux development VM. It keeps
project toolchains, system packages, and coding-agent execution inside the guest
while you work through VS Code, a terminal, explicit commands, or any ssh-enabled orchestrator like herdr and alike. 
## Why VBX exists

Working across projects often means installing different runtimes, compilers,
command-line tools, and supporting services. Installing everything on the host
mixes those dependencies into one machine. Setting up separate VMs by hand adds
repeated work: choosing an image, mounting source directories, installing tools,
configuring SSH, and connecting the editor.

VBX packages that setup into a project-level workflow:

- **Give each project its own development machine.** Each project directory maps
  to a separate Ubuntu VM with its own installed packages, home directory, and
  guest-local state. Projects with the same directory name still get distinct VMs.
- **Keep source files on the host.** VBX mounts your working tree directly into
  the guest. Host tools and guest tools see the same files, with no separate clone
  or synchronization step. Removing the VM leaves the host working tree intact.
- **Use the same machine from your editor and shell.** VS Code Remote SSH,
  interactive terminals, explicit commands, and coding agents share the guest's
  files and tools.
- **Describe setup once.** Local defaults and project configuration select
  resources, additional mounts, and guest packages. Home overlays, environment
  files, and setup scripts supply personal and project needs.
- **Keep the environment between sessions.** Closing the editor or terminal
  leaves the VM available. Stopping and starting it preserves installed tools
  and guest-local files. Normal reuse does not repeat provisioning.

## Isolated coding-agent runs

Coding agents running in YOLO or automatic-approval mode can execute commands,
install packages, and change their environment with little supervision. VBX gives
that work a separate Linux VM with its own kernel, system packages, and home
directory. The agent's shell commands and processes execute inside the guest.
Your host OS and its installed toolchains stay outside that guest environment.

Install and start your chosen agent inside the VM, using its own approval
settings. VBX does not choose those settings or start an agent automatically.
You can inspect the same working tree through your editor or terminal, keep the
VM between sessions, or discard it and create a fresh guest environment.

The isolation boundary still includes what you expose to the guest:

- Writable host mounts let the agent modify those host files, including the
  project. Use a dedicated worktree or disposable copy for unattended changes.
- Credentials supplied to the guest are available to its programs. Provide only
  the credentials the task needs, and mount only the required directories.
- Guest networking is unrestricted by VBX. This is not a network-isolation policy.
- Recreating the VM resets guest-local state. It does not undo edits to mounted
  host files or revoke credentials.

## How it works

From a project directory, run `vbx --vscode`, `vbx --terminal`, or
`vbx --workspace-only`. On first use, VBX creates an Ubuntu 26.04 LTS minimal VM
through Lima and QEMU, mounts the project under `/workspaces/`, applies the
selected setup, and opens the requested interface after setup succeeds.
Subsequent calls reconnect to that project's VM.

The guest is a full Linux system with its own kernel, systemd, and package
manager. Tools install inside the guest, using guest sudo where needed. The host
runs Lima/QEMU and your chosen client tools.

## Operating model

- Setup runs at creation time. Configuration edits take effect through explicit
  removal and recreation; VBX does not reconcile an existing environment.
- Failed creation keeps a disposable VM for inspection and prints the commands
  to remove and recreate it after correcting the error.
- Fresh creation resolves packages from their repositories. VBX captures setup
  inputs for retries; it does not lock the entire dependency tree or promise
  byte-identical rebuilds.
- Setup scripts and installed tools are trusted code. Guest programs can access
  configured host mounts and available credentials. A writable project mount
  lets guest programs modify that host working tree.

The supported host platform is local Linux x86_64 with KVM acceleration. The
steps below cover installation and daily use.

## 1. Check prerequisites

Run VBX on a **Linux x86_64 host**, as your normal user (without sudo).

| Requirement | Purpose |
| --- | --- |
| Hardware virtualization and read/write access to `/dev/kvm` | Run the guest with KVM acceleration |
| Latest stable Lima (`limactl`) | Manage the VM and its SSH connection |
| QEMU (`qemu-system-x86_64`, `qemu-img`) | Run the VM and manage its disk |
| OpenSSH client (`ssh`) | Connect to the guest |
| Bash, Perl with core `JSON::PP`, `flock`, `tar`, and standard GNU/Linux utilities | Run the launcher and provisioning transport |
| Host and guest internet access during creation | Download the Ubuntu image and install packages |

Allow enough host RAM for the guest (4 GiB by default) plus the host OS. The
30 GiB default guest disk is sparse and consumes host storage as it grows.

**VS Code with Remote - SSH is optional**, needed only for `--vscode` (the default
launch mode). Terminal and command workflows do not require it. Development
runtimes install inside the guest. Docker and libvirt are not host requirements.

Follow [host setup](docs/manuals/host-setup.md) to install missing prerequisites
or grant KVM access. Reuse an existing working QEMU/KVM installation.

## 2. Install the launcher

Run on your Linux x86_64 host, from this checkout:

```sh
mkdir -p "$HOME/.local/bin"
ln -s "$PWD/vbx" "$HOME/.local/bin/vbx"
export PATH="$HOME/.local/bin:$PATH"
vbx --check
```

If the check fails, follow [host setup](docs/manuals/host-setup.md), then rerun it.

## 3. Configure defaults

```sh
mkdir -p config.local
cp -rn config.local.example/. config.local/
```

Edit `config.local/config.yaml` for your guest resources and tools. The example
selects Git and compiler tools through `apt.packages`. Add the Ubuntu packages
your projects need.

Follow [provisioning](docs/manuals/provisioning.md) to install tools, add API keys,
copy personal home files, or run project setup. Existing VMs keep their original
tools and configuration. See [configuration](docs/manuals/configuration.md) for
resources and mounts.

## 4. Enable VS Code access (optional)

Install **Remote - SSH** (`ms-vscode-remote.remote-ssh`) in host VS Code.

```sh
vbx --ssh-config
```

Add the printed `Include` line near the beginning of `~/.ssh/config`.

## 5. Open a project

```sh
cd /path/to/project
vbx --vscode
```

Open a remote terminal to build and run your product. Project files appear under
`/workspaces/`. The guest account is `dev`, with home `/home/dev`.
No agent starts automatically.

## 6. Other launch modes

```sh
vbx --terminal
vbx --workspace-only
vbx --exec git status
```

If creation fails, correct its inputs and recreate the disposable VM:

```sh
vbx --remove
vbx --workspace-only
```

VBX keeps the incomplete VM for inspection until you remove it. Host project
files remain intact. Retrying without removal uses the original saved inputs.

## 7. Stop or remove the VM

Run from the project directory:

```sh
vbx --stop
vbx --remove
```

Removal requires confirmation and deletes guest-local files. Host project files remain.

- [Command reference](docs/manuals/commands.md)
- [Configuration](docs/manuals/configuration.md)
- [Provisioning and credentials](docs/manuals/provisioning.md)
- [Self-contained recipes](docs/manuals/recipes.md)
- [Release notes](CHANGELOG.md)
- [Documentation index](docs/README.md)

## License

[MIT](LICENSE). Copyright (c) 2026 VBX contributors.
