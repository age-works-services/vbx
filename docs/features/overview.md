# Release overview

VBX provides persistent, per-project Ubuntu development VMs on local Linux x86_64
hosts. It keeps project toolchains and coding-agent execution inside a guest OS,
with shared access through the editor and terminal. It also supports YOLO-style
agent workflows where shell commands and package installs run in a separate VM.
Writable host mounts, supplied credentials, and guest networking remain explicit
parts of the exposure boundary.

Lima manages QEMU directly; these VMs do not belong to libvirt and do not appear
in virt-manager.

## Runtime

- Guest: official Ubuntu 26.04 LTS minimal, QEMU x86_64, containerd disabled.
- Defaults: 2 CPUs, 4 GiB memory, and a sparse disk with 30 GiB capacity.
- Account: `dev`, home `/home/dev`, numeric UID mapped from the host user.
- Workspace: direct 9p mounts, with a writable primary project under
  `/workspaces/<project-slug>` and optional additional directories.
- Interfaces: VS Code Remote SSH, login shell, explicit command, or background VM.
- Host tooling: Bash, core Perl/JSON::PP, standard Linux utilities, SSH, Lima,
  QEMU, and KVM. No Python or external YAML parser is required by the launcher.

VM names combine a readable slug with a hash of the canonical project path.
Different paths with the same basename produce separate instances. The guest
home and installed packages persist independently of mounted host project files.

## Configuration and creation

Built-in defaults, checkout-local settings, and project settings form ordered
layers. Scalars override; list entries append, with documented deduplication.
The restricted YAML parser rejects unsupported constructs and treats values as data.

Creation captures selected module directories, APT requirements, environment,
home overlays, and project setup. Project-controlled inputs require approval.
The guest runs base provisioning, combined APT installation, modules, home
overlays, environment loading, and project setup before the requested tool opens.

Provisioning recipes are self-contained. Users explicitly select prerequisites
before consumers. No recipe installs another recipe implicitly. Tool-specific
selection and installation details live in the [recipe manual](../manuals/recipes.md).

## Persistence and recovery

Ready instances reuse their saved configuration without provisioning again.
Changes to resources, mounts, tools, environment, or home inputs take effect
through explicit removal/recreation.

Failed creation blocks launch and retains the disposable VM for inspection.
The launcher prints remove/recreate commands. Ordinary retries retain captured
inputs and skip successful setup steps; they do not apply edited configuration.
There is no setup refresh, reconciliation, rollback, or automatic deletion.

Per-project locks serialize lifecycle changes. The launcher releases its lock on
exit and excludes the lock descriptor from Lima child processes.

## Trust boundaries

Modules and project setup are trusted code with guest sudo. Guest processes can
access the configured host mounts and guest credentials. npm lifecycle approvals
constrain installation scripts; they do not sandbox extensions or certify packages.
The browser uses Chromium's sandbox with a guest-local AppArmor user-namespace rule.

VBX does not manage host packages during normal operation, alter libvirt, adopt
other environments, or modify `pi-run`. Upstream package vulnerability remediation
is outside this release's scope.

## References

- [Configuration](../manuals/configuration.md)
- [Provisioning](../manuals/provisioning.md)
- [Recipe contract](../manuals/recipes.md)
- [Provisioning design](provisioning-design.md)
- [Validation](../testing/validation.md)
