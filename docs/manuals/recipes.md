# Creation-only recipes

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
| `dev-tools` | General development tools and GitHub CLI completion | None |
| `node` | NodeSource bootstrap, Node/npm, user npm prefix and policy | None |
| `pi` | Git, Pi CLI, subagents, prompt templates, CLI script approvals | Node/npm |
| `pi-web-fetch` | Extension, Python/pipx, browser libraries, Chrome, AppArmor rule, rendering check | Pi |

There is no standalone browser module or automatic dependency expansion. Pi does
not install Node, Python, or Chrome implicitly. A compatible preinstalled toolchain
can satisfy prerequisites without selecting `node`.

## Self-contained module files

```text
modules/<name>/
  apt.txt          # Optional OS package requirements
  setup.sh         # Required, blocking creation entrypoint
  ...              # Local helpers, policies, assets, and checks
```

All installer code stays in its owning module directory. Modules do not load
installer code from the framework or sibling modules. The core provides generic
transport, APT resolution, ordering, markers, and setup-wide npm enforcement.

The closed registry in `scripts/config.pl` accepts bundled names only. Adding a
bundled module requires registering its identifier there. Project-local modules,
remote loading, provider selection, and dependency graphs are unsupported.
Use `.vbx/setup.sh` for project-specific commands.

## APT requirements

`apt.txt` contains one requirement per line. Blank lines and full-line comments
work. Requirements use package names (optionally `:architecture`) and an optional
single `>=`, `<=`, or `=` Debian version comparison:

```text
# Packages from configured guest repositories
curl
git
jq (>= 1.7)
```

VBX combines these with `apt.packages`, deduplicates identical requirements, and
retains distinct constraints. APT resolves them with `apt-get satisfy --no-remove
--no-install-recommends`. Alternatives, URLs, flags, and arbitrary APT directives
are unsupported. Conflicts and unavailable versions stop setup.

A module owns external repository bootstrap in its setup script. The `node`
module uses the signed NodeSource Node 24 repository if compatible Node/npm are
absent. Do not also request incompatible Ubuntu Node/npm packages in `apt`.

Node itself is system-installed through APT. npm global packages use
`~/.local/lib/node_modules`, with commands in `~/.local/bin`.

## Execution and failure

VBX captures complete selected module directories, requirements, environment,
home overlays, and project setup before creation. The combined APT phase runs
before modules. Each `setup.sh` runs as `dev` from the primary project mount and
uses `sudo -n` explicitly for privileged guest operations.

Scripts should use `set -euo pipefail`, tolerate partial execution, locate local
files through their own script path, and check functionality before success.
Completed-step markers support ordinary retries of captured inputs.

Failure blocks launch and keeps the disposable VM for inspection. Correct the
inputs, then use `vbx --remove` and `vbx --workspace-only`. There is no in-place
setup refresh, rollback, update hook, uninstall hook, or reconciliation.
Ready VMs retain their original setup.

## npm lifecycle policy

npm >=11.19.0 is required; the policy fixtures use 11.19.1. Modules configure
the user prefix and strict lifecycle enforcement. Project setup inherits the
same enforcement. Unknown scripts and failed approved scripts stop provisioning.

Policies use native npm `allowScripts` entries. VBX does not automatically
approve discovered scripts, skip lifecycle scripts, or allow all scripts.

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

Modules, project setup, and installed extensions are trusted code with guest-user
permissions and guest sudo. They can access configured host mounts and available
credentials. Lifecycle enforcement is not a sandbox against explicitly approved
code or override flags supplied by trusted setup code.

Exact-version script approval is separate from package security analysis. A
recorded Puppeteer 24.43.1 audit contained high-severity transitive findings.
It did not assess native Chrome vulnerabilities. The browser sandbox does not
sandbox npm installation.

VBX does not certify upstream packages as vulnerability-free or maintain their
security fixes. Upstream vulnerability remediation is outside this release's
scope. See [validation](../testing/validation.md#package-review-observations) for
the recorded audit context.
