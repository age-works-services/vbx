# Project overview

- VBX gives each project a persistent Ubuntu development VM through Lima and QEMU/KVM on Linux x86_64.
- Host source files mount into the guest under `/workspaces/`. Users connect through VS Code Remote SSH, a terminal, or `vbx --exec`.

## Code map

- `vbx`: Bash CLI for VM identity, lifecycle, configuration rendering, and launch modes.
- `scripts/`: host checks, provisioning, Perl configuration/environment parsing, and package policies.
- `lima/default.yaml`: base VM template.
- `modules/`: optional development tools, Node.js, Pi, and Pi web-fetch setup.
- `config.local.example/`: example local defaults. Project overrides live in `.vbx/`.
- `docs/`: manuals, design notes, and validation procedures. Start with `README.md`.
- `test/`: shell test suites using mocked external tools.

## Working constraints

- Setup runs at VM creation. Configuration changes require explicit removal and recreation.
- VM removal preserves host files. Writable mounts and supplied credentials remain accessible to guest programs.
- Treat `config.local/` as private configuration. Do not expose secrets.
- Run relevant `bash test/test-*.sh` suites after code changes. See `docs/testing/validation.md` for prerequisites and real-VM checks.
