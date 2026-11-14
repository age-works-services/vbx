# Commands

Run VBX from the project directory, or select it explicitly:

```sh
vbx --project /path/to/project --vscode
```

Only one action may be selected. With no action, VBX uses the saved
`launch.default` setting, which defaults to VS Code.

## Start and connect

| Command | Behavior |
| --- | --- |
| `vbx` | Create/start/reuse the VM and perform its default launch action |
| `vbx --vscode` | Open the configured guest folder through VS Code Remote SSH |
| `vbx --terminal` | Open a guest login shell in the primary project mount |
| `vbx --workspace-only` | Create/start/reuse the VM without opening a tool |
| `vbx --exec COMMAND...` | Run a command in the guest from the primary project mount |

Pi starts only when requested, for example `vbx --exec pi`.
The primary project is mounted at `/workspaces/<project-slug>`.

Everything following `--exec` belongs to the guest command. Place `--project`
before `--exec`. Arguments retain their boundaries and are not evaluated as host
shell code. Use an explicit guest shell for pipelines or multiple commands:

```sh
vbx --exec printf '%s\n' 'a value with spaces'
vbx --exec bash -c 'make && ./app'
```

Managed commands load the captured guest environment and prepend `~/.local/bin`
to PATH. Login and interactive Bash sessions also integrate that environment.

## Stop and remove

```sh
vbx --stop
vbx --remove
```

Stopping preserves guest-local files and installed tools. Removal requires an
interactive terminal and confirmation by typing the VM name. It deletes the VM
and its VBX state, while retaining mounted host project files. Preserve any
guest-local data you need before removing a ready environment.

To apply changed configuration, remove and create again. For failed creation,
treat the incomplete VM as disposable and follow the printed recovery commands.

## Inspect and configure

| Command | Behavior |
| --- | --- |
| `vbx --check` | Check host prerequisites |
| `vbx --render` | Validate desired configuration and print Lima YAML without starting a VM |
| `vbx --print-name` | Print the canonical project VM name |
| `vbx --ssh-config` | Print the SSH Include line; VBX does not edit host SSH configuration |
| `vbx --help` | Print CLI usage |

`--render` reads current configuration even if a VM already exists. Normal reuse
uses the saved configuration instead. Rendering is not a full guest installation
test. See [configuration](configuration.md) for supported settings.

For direct guest access and host inspection:

```sh
limactl list
limactl shell "$(vbx --print-name)"
```

These commands use Lima directly. They are useful when inspecting a failed VM
without retrying VBX setup.
