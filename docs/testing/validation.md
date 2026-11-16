# Testing and validation

## Automated tests

Use Go 1.25 or later. Race checks require cgo and a C compiler. Guest fixtures
require Bash and standard Linux utilities. The automated suite covers VBX's
CLI, lifecycle, five-layer settings precedence, shared inline/sidecar persistence
schemas, bundling, provisioning orchestration, and installation. The guest transport fixture exposes only its required shell
utilities.

From the checkout:

```sh
make build test check cross-check
```

Or run individual checks:

```sh
go test ./...
go vet ./...
CGO_ENABLED=1 go test -race ./...
bash test/test-host.sh
bash test/test-vbx.sh
bash test/test-provision.sh
bash test/test-apt.sh
bash test/test-make.sh
bash test/test-persistence.sh
bash modules/aws-cli/test/test-aws-cli.sh
```

Go tests use temporary HOME/XDG paths, injected Lima/editor services, and local
Bash transport fixtures. Guest shell tests intercept APT and sudo and use
synthetic modules to exercise VBX's provisioning sequence and retry behavior.
Tests must not modify host packages, host `/etc`, private configuration, or real
VMs. Module-specific tests live under their module directories. `make test` also
runs `modules/*/test/test-*.sh`; the AWS CLI offline suite requires GnuPG.
Cross-builds check compilation only; VM operation requires Linux x86_64/KVM.

## Linux VM checks

Use a disposable project. Follow the [README](../../README.md) for host setup,
installation, configuration, and editor access.

1. Run `vbx check-host-prerequisites`, then create the VM with
   `vbx start --workspace-only`. Include an empty-module/package configuration.
2. Verify account `dev`, HOME `/home/dev`, and the primary mount under
   `/workspaces/`. Confirm writes in both directions between host and guest.
3. Check configured additional mounts, read-only behavior, home-overlay precedence,
   file modes, and literal environment values. Use synthetic values, not credentials.
4. Test `exec` with spaces, literal shell syntax, guest flags resembling VBX options,
   and nonzero exit codes. Check general options before and after the command.
5. Open `start --terminal`. Confirm a TTY, Ctrl+C interrupts a command without
   ending the guest shell, and `exit` returns to the host.
6. Open `start --vscode`. Verify the remote workspace and concurrent CLI access.
7. Stop and restart. Check guest-home persistence and that provisioning does not
   repeat. Confirm bare `vbx` and `vbx start` use the same configured target.
8. Force project setup to fail. Check launch blocking, retained captured inputs,
   lock release, and retries that keep using the captured script after host edits.
9. Interrupt lifecycle operations with SIGINT, SIGTERM, and SIGHUP. Check child
   cancellation, temporary-file cleanup, and lock release. Retained setup archives
   in private instance state support retries and must not appear in workspace mounts.
10. Confirm removal asks for the exact VM name and preserves host project files.
11. Delete a disposable VM directly through Lima, leaving VBX state behind. Check
    that startup refuses automatic repair and confirmed `vbx remove` clears the state.

Do not export temporary XDG settings into your normal shell. If using isolated
configuration for CLI tests, launch VS Code with its normal configuration so its
settings and extensions remain available.

## Persistence VM checks

Use a disposable VM and synthetic session/settings data. Select both
persistence modules described in [project module persistence](../manuals/persistence.md).

1. Verify `findmnt` reports the captured `mountN` source and `9p` filesystem at
   every `/mnt/vbx-data/<module>` mount, with writable mode and matching host UID.
2. Verify guest-created private directories retain mode 0700 on the host.
3. Open VS Code after the mappings exist. Install, update, and remove a remote
   extension and change a remote machine setting. Verify file watching and an
   extension with native code through the mounted directory.
4. Create a Pi session. Remove and recreate the VM for the same project/data-home
   base. Verify session resumption, remote extension availability, and settings.
5. Force project setup to fail, then retry. Verify installer markers skip repeated
   installation and mappings remain connected after overlays are reapplied.
6. Verify overlay collisions fail before VM creation and populated guest
   directories fail without losing either copy. Use direct Lima shell access
   to inspect conflicts.
7. Remove a captured host source while the VM is stopped. Confirm VBX refuses
   startup before Lima can recreate an empty source. Restore the source.
8. Replace a guest link with a real directory. Verify readiness blocks launch and
   provides recovery instructions that preserve the guest-local copy.
9. Verify retained-data deletion refuses a stopped VM. Remove the VM, confirm
   the project identifier, and verify only that project's retained data is deleted.

## Module checks

Select prerequisites explicitly and test each selected module in a disposable VM.
For Pi-equipped guests:

```sh
vbx exec node --version
vbx exec npm --version
vbx exec npm config get strict-allow-scripts
vbx exec pi --version
vbx exec pi list
vbx exec pi mcp list
```

The toolchain must meet module requirements. npm strict policy should print
`true`, and Pi should list the selected extensions.

With `pi-web-fetch` selected, verify its creation-time sandboxed rendering check
and an explicit Pi `web_fetch` call, including Python extraction. With an API
integration configured, verify a harmless authenticated query without printing
credentials.
