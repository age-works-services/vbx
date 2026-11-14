# Changelog

## First release

### Supported environment

- Local Linux x86_64 hosts using Lima, QEMU, and KVM.
- Ubuntu 26.04 LTS minimal guests, with account `dev` and host-derived numeric UID.
- Persistent VM identity based on the canonical project path.

### Launcher

- VS Code Remote SSH, interactive terminals, explicit commands, and background startup.
- Stop and confirmed removal, preserving host project files.
- Local/project configuration layers, resource settings, and additional 9p mounts.
- Per-project operation locks that do not leak into persistent Lima children.

### Provisioning

- Creation-time guest update/upgrade and native APT package requirements.
- Self-contained `node`, `pi`, `pi-web-fetch`, and `dev-tools` modules.
- Explicit module order and strict, version-scoped npm lifecycle approvals.
- Personal/project home overlays, literal environment files, and project setup scripts.
- Captured setup inputs, completion markers, and launch blocking on failure.
- Failed-creation diagnostics with explicit remove/recreate commands.

### Release boundaries

- Setup runs during creation. Configuration changes require removal/recreation.
- Project-local module definitions and remote module loading are unsupported.
- Pi and web-fetch are optional. VBX never launches Pi automatically.
- VM isolation and lifecycle approvals do not establish upstream package security.
  Upstream vulnerability remediation is outside VBX's scope.
- The launcher does not modify existing `pi-run` environments.

See [validation](docs/testing/validation.md) for evidence and reproducible checks.
