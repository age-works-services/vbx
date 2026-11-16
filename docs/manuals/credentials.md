# Environment and credentials

For private repositories, see [Git authentication](#git-authentication).
For API keys and other environment variables, use the steps below.

## Supply environment variables

Set up a private global configuration directory (no source checkout required):

```sh
config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/vbx"
mkdir -p "$config_dir"
chmod 700 "$config_dir"
(umask 077; touch "$config_dir/.env")
chmod 600 "$config_dir/.env"
```

Edit `$config_dir/.env` with your editor. Add only needed variables, for example:

```dotenv
ANTHROPIC_API_KEY=your-key-here
```

If `$config_dir/config.yaml` does not exist, create it with:

```yaml
version: 1
envFiles:
  - .env
```

For an existing configuration, add `.env` to its `envFiles` list, or add that
setting if absent. Keep the existing `version: 1` and other settings.
The checkout's `config.local.example/env.example` lists more optional keys.
Pi can also use its normal interactive login.

Keep the private directory outside your source checkout. Never put keys in
committed YAML or home examples.

## Git authentication

Host Git credentials and your host SSH agent are not part of VBX's environment-file
configuration. Choose authentication for commands running in the guest:

- For HTTPS, authenticate inside the guest with your provider's CLI or a suitable
  Git credential helper. For GitHub, select `git` and `gh` through `apt.packages`,
  then run `gh auth login` and `gh auth setup-git` inside the guest. Avoid tokens
  in remote URLs, shell history, or committed configuration.
- For SSH, prefer a repository-scoped guest key or explicitly forward your host
  agent for a trusted session. VBX has no agent-forwarding option. Do not assume
  `vbx exec`, terminal launches, and VS Code sessions share a forwarded agent.

To forward an existing host agent for one SSH session, first configure
[SSH access](commands.md#ssh-access). On the host:

```sh
ssh-add -l
vbx start --workspace-only
ssh -A "lima-$(vbx name)"
```

`ssh-add -l` should list the intended host key. Inside that SSH session, use
`ssh-add -l` to check forwarding, then run Git commands from the project mount.
Ensure the guest has `git` and `openssh-client` (select them through `apt.packages`
before creation if needed). Forwarding applies to that session. Closing it ends
that forwarded connection.

Guest programs with access to the forwarded socket can authenticate using your
host agent while connected. Forward only to trusted guests and load only needed
keys. Avoid copying your entire host `.ssh/` directory into a home overlay.

Guest-local login state and private keys survive stopping but disappear on VM
removal unless explicitly retained. Keep retained credentials private, and revoke
credentials separately when necessary. VM removal does not revoke them.

## File syntax and precedence

Environment files use single-line `NAME=VALUE` entries. Blank lines, comments,
optional `export`, empty values, and surrounding single/double quotes work.
Quotes wrap literal text. There is no variable expansion, command execution,
escape processing, or multiline syntax. Use single quotes for values containing
backslashes. An unquoted whitespace-prefixed `#` begins an inline comment.

Environment files load in merged list order (see [configuration layers](configuration.md#files-and-precedence)).
Later definitions win. Relative global paths resolve from `$config_dir` and
relative project paths resolve from the project root. Each selected file must
exist. `envFiles: []` adds no files and does not clear inherited entries.

## Guest environment and security

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

Environment changes follow the [creation-time configuration model](configuration.md#reuse-and-configuration-changes).
See [trust and approval](provisioning.md#trust-and-project-approval) before sharing keys with a project.
