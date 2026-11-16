# Modules

Select bundled tools before creating a VM. Module scripts run during creation
or retries. See [provisioning](provisioning.md) for project setup and
[applying changes](configuration.md#reuse-and-configuration-changes) for existing VMs.

On this page: [Selection](#selection) · [Tool notes](#tool-notes) ·
[APT requirements](#apt-requirements) · [npm lifecycle policy](#npm-lifecycle-policy) ·
[Pi preferences](#pi-preferences)

## Selection

Select bundled modules in local or project configuration:

```yaml
version: 1
apt:
  packages:
    - build-essential
    - gh
    - 'jq (>= 1.7)'
modules:
  - node
  - pi
  # - pi-web-fetch
```

Lists append across layers. Duplicate module names run once, at their first
position. Select prerequisites before consumers, in the same or an earlier layer.

| Module | Owns | Prerequisite |
| --- | --- | --- |
| `aws-cli` | AWS CLI v2 from AWS, with archive signature verification | None |
| `dev-tools` | General development tools and GitHub CLI completion | None |
| `node` | NodeSource bootstrap, Node/npm, user npm prefix and policy | None |
| `pi` | Git, Pi CLI, subagents, prompt templates, CLI script approvals | Node/npm |
| `pi-web-fetch` | Extension, Python/pipx, browser libraries, Chrome, AppArmor rule, rendering check | Pi |
| `pi-project-persistent-sessions` | Project Pi session persistence | Pi |
| `vscode-project-persistent-state` | Project remote extension and machine-settings persistence | VS Code Remote SSH |

Select prerequisites explicitly. A compatible preinstalled toolchain can satisfy
the Node/npm prerequisite.

## Tool notes

### General development tools

`dev-tools` installs compiler/Make tools, Git/GH, Python/venv/pipx, archive tools,
editors, and network/process diagnostics. It also enables Bash completion for
`gh`. See the [exact package list](../../modules/dev-tools/apt.txt).
Select `postgresql-client` separately through `apt.packages` if needed.

### Node

`node` accepts Node >=22.19 and npm >=11.19. Otherwise it bootstraps NodeSource
Node 24 LTS. Node installs system-wide through guest APT. npm global commands
use the guest user's `~/.local/bin`. Avoid incompatible Ubuntu Node/npm packages
in `apt.packages`.

### Pi and web extraction

Select `node` then `pi` for Pi CLI, subagents, and prompt templates.
For web extraction, add `pi-web-fetch` after `pi`:

```yaml
modules:
  - node
  - pi
  - pi-web-fetch
```

Place these keys in a `config.yaml` with `version: 1`. Pi supports interactive
login and [API-key environment files](credentials.md). After creation:

```sh
vbx exec pi --version
vbx exec pi list
vbx exec pi mcp list
vbx exec pi
```

Pi starts only when requested. `pi-web-fetch` installs Python/pipx,
Puppeteer/Chrome, and an extension. It checks sandboxed rendering and, on Ubuntu's
AppArmor-enabled kernel, grants user namespaces specifically to Chrome in the
Puppeteer cache. It leaves the global user-namespace policy unchanged.
See [npm lifecycle policy](#npm-lifecycle-policy) for reviewed script versions.

### Retained sessions and editor state

The persistence modules are separate from tool installation. See
[project persistence](persistence.md) for storage, conflicts, and deletion.

## Module implementation

For module files, registration, and installer contracts, see
[developing bundled modules](../module-development.md).
Use `.vbx/setup.sh` for project-specific installation commands.

The [AWS CLI module guide](../../modules/aws-cli/README.md) covers its signed
installer and update behavior.

## APT requirements

Use `apt.packages` in global or project configuration for Ubuntu packages:

```yaml
version: 1
apt:
  packages:
    - git
    - 'jq (>= 1.7)'
```

Requirements accept package names (optionally `:architecture`) and one optional
`>=`, `<=`, or `=` Debian version constraint. Alternatives, URLs, flags, and
arbitrary APT directives are unsupported.

VBX combines requirements from selected modules and configuration layers.
Inspect them with `vbx show-project-effective-setup`. APT resolves the combined
list without removing packages or installing recommendations. Conflicts and
unavailable versions stop setup. Avoid Ubuntu Node/npm requirements that conflict
with the `node` module.

For module-owned `apt.txt` files and repository bootstrap, see
[module development](../module-development.md#apt-requirements).

## npm lifecycle policy

npm >=11.19.0 is required for strict lifecycle enforcement. Modules configure
the user prefix and strict lifecycle enforcement. Project setup inherits the
same enforcement. Unknown scripts and failed approved scripts stop provisioning.

Policies use native npm `allowScripts` entries with explicit approvals.

| Policy | Exact reviewed approvals |
| --- | --- |
| `modules/pi/core-npm-policy.json` | `@google/genai@2.21.0` echo/no-op preinstall; `protobufjs@7.6.6` dependency-warning postinstall |
| `modules/pi-web-fetch/npm-policy.json` | `puppeteer@24.43.1` browser-installation postinstall |

Global Pi installation passes an explicit `--allow-scripts` list. The web-fetch
module merges its approval into `~/.pi/agent/npm/package.json#allowScripts`,
preserving unrelated policy and dependency entries. npm project installs read
that manifest instead of a CLI allowlist.

npm resolves web-fetch's Puppeteer dependency. A different script-bearing version
fails until covered by a reviewed policy. Coordinate dependency/policy changes
and recreate the VM to adopt them. The module checks rendering through web-fetch's
actual Puppeteer installation, with Chromium's sandbox enabled.

## Pi preferences

Pi's own `settings.json#packages` can install extensions independently of VBX.
Keep it consistent with selected modules, particularly when opting out of web-fetch.
The shipped `settings.json.example` is reference-only and Pi does not load it.

Home overlays replace JSON files whole. When supplying a real `settings.json`,
preserve the package registrations needed by your selected modules.

## Security notes

Review the [trust boundary and project approval](provisioning.md#trust-and-project-approval)
before applying setup. Lifecycle approvals do not restrict installed programs.
