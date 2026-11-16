# Install, update, and uninstall

## Install from source

Follow the [README quickstart](../../README.md#quickstart). Build with Go 1.25 or
later and Make. `make install` writes the standalone executable to
`~/.local/bin/vbx` and does not require Go at runtime. Run `make help` for path
overrides. Bare `make` builds, tests, and installs in that order.

The supported VM host is local Linux x86_64 with KVM. Cross-compilation checks
for other platforms do not establish runtime support.

## Update

Update your source checkout to the revision you intend to use, review its changes,
then run:

```sh
make install
vbx --help
vbx check-host-prerequisites
```

Reinstallation replaces the executable and its embedded modules and scripts.
It leaves global configuration, VMs, and retained data in place. Existing VMs
keep their saved setup. [Back up guest-local files](persistence.md#back-up-guest-local-files) before
[recreating](configuration.md#reuse-and-configuration-changes) to adopt new setup.
VBX currently has no version command or `--version` flag. Record the source
revision with `git rev-parse HEAD` in the checkout when installing.

## Uninstall

1. Run `vbx list-vms`. For each VM you want to discard, preserve guest-local work
   and run `vbx remove --project DIR` while the launcher is still installed.
   For a missing directory, follow [moved/deleted project recovery](troubleshooting.md#moved-or-deleted-projects).
2. Delete retained data only if unwanted, using `vbx remove-project-data` (or an
   explicit saved project identifier). See [data deletion](persistence.md#explicit-data-deletion).
3. Remove `~/.local/bin/vbx`, or the executable at your custom installation path.
4. Remove the VBX `Include` line you added to `~/.ssh/config`.
5. Optionally remove unused VBX configuration and state after checking the
   [managed locations](configuration.md#files-vbx-manages). They may contain secrets.

Keep Lima and QEMU if other VMs use them. Uninstalling the executable alone leaves
VMs and data intact.

## Networking

Creation requires network access for the image, APT, and selected module downloads.
VBX provides no offline installer, proxy configuration key, or port-forwarding
command. Configure those capabilities through the guest, Lima, and your SSH
client as appropriate. Do not assume services in the guest are unreachable from
the host or other networks. Check Lima's effective forwarding behavior for your
installation. Use SSH local forwarding when you need an explicit connection:
see [SSH access](commands.md#ssh-access).
