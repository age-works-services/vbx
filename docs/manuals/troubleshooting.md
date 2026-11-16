# Troubleshooting

## Find the failure

| Symptom | Next step |
| --- | --- |
| Missing Lima, QEMU, SSH, or KVM access | Run `vbx check-host-prerequisites`, then follow [host setup](host-setup.md) |
| Setup fails or launch is blocked | Read the terminal output, then [inspect failed setup](#failed-setup) |
| Edited configuration has no effect | [Recreate](configuration.md#reuse-and-configuration-changes) |
| Pi reports missing Node | Put `node` before `pi` in the layer selecting Pi, then recreate |
| npm rejects an installation script | Check the package/version against [lifecycle policy](recipes.md#npm-lifecycle-policy); review before approving changes |
| VS Code cannot connect | Check `code` in PATH, Remote - SSH installation, and the SSH Include line from `vbx show-host-ssh-config-line` |
| VS Code uses old environment values | Restart its remote server; see [credentials](credentials.md) |
| Persistence refuses a populated directory or link | [Resolve conflicts](persistence.md#conflicts-and-existing-data) before removing anything |
| Project directory moved or deleted | [Recover files and clean up the old VM](#moved-or-deleted-projects) |
| VM/state mismatch | See [state mismatches](#state-mismatches) |
| Operation remains busy | See [operation locks](#operation-locks) |
| Private Git repository rejects authentication | See [Git authentication](credentials.md#git-authentication) |
| Problem remains unresolved | Collect a [diagnostic report](#report-an-unresolved-problem) |

## Failed setup

VBX streams setup output to the invoking terminal. It blocks launch after failure
and retains the incomplete VM and captured inputs. Inspect without retrying setup:

```sh
limactl shell --workdir=/ "$(vbx name)"
```

Inside the guest, inspect provisioning markers under
`~/.local/state/vbx/provision/`. Markers indicate completed steps, rather than
providing a transcript. VBX has no dedicated provisioning-log command. If you
capture terminal output yourself, treat it as sensitive because user scripts may
print credentials.

An ordinary `vbx start` retries the captured inputs and skips successful steps.
This can resolve a transient network failure. Edited configuration and scripts do
not replace those inputs.

To adopt a correction, first [back up guest-local files](persistence.md#back-up-guest-local-files),
then run:

```sh
vbx remove
vbx start --workspace-only
```

Removal preserves mounted host files and declared retained data. Module ordering
spans configuration layers. Adding Node in a later layer cannot move it before an
inherited Pi selection. See [module selection](recipes.md#selection).

## State mismatches

`vbx status` reports Lima runtime state, which does not indicate setup completion.
Use `limactl list` to compare the actual instances.

- If Lima has no VM but VBX still has instance state, use `vbx remove` to confirm
  discarding orphaned state before creating again.
- If an existing VM has a missing or invalid VBX snapshot, stop normal startup.
  Preserve valuable guest-local files through direct Lima access first.
- Use `vbx remove` for normal deletion. Avoid deleting state directories by hand.
  Direct Lima deletion leaves VBX state behind and still requires explicit cleanup.

## Moved or deleted projects

Moving a project selects a new VM. The old VM, VBX state, and retained data remain
under the old identity. VBX has no remove-by-VM-name option for VM removal.

Use this recovery path to keep normal VBX locking and cleanup:

1. Run `vbx list-vms` and record the old VM's exact `NAME` and `PROJECT` path.
   If the path is unavailable (`-`), recover it from your own records before
   continuing. Do not guess a directory from the VM's shortened name.
2. Restore the original directory path. For cleanup alone, an empty directory at
   that path is sufficient. Do not overwrite an existing directory. For example:

   ```sh
   original=/absolute/original/project
   mkdir -p "$original"
   vbx name --project "$original"
   ```

   Compare the printed name with the recorded VM name. Continue only if they
   match exactly. Symlinked parent directories can affect the canonical path.
3. Preserve valuable guest-local work using the [backup procedure](persistence.md#back-up-guest-local-files),
   substituting the recorded VM name. A stopped guest may need its original host
   mount sources restored before Lima can start it. An empty project directory
   does not restore the project's files or any additional mounts.
4. Remove the old VM and its VBX state and SSH link:

   ```sh
   vbx remove --project "$original"
   ```

   VBX stops a running VM before deleting it. If Lima already has no VM, this
   command can still discard orphaned VBX state after confirmation.
5. Keep retained data until you have recovered anything valuable. To discard it,
   run `vbx remove-project-data OLD-VM-NAME` with the exact recorded identifier
   and the original `XDG_DATA_HOME` setting. See [data deletion](persistence.md#explicit-data-deletion).

Remove any empty directories you created for recovery only after cleanup. Do not
start VBX against a placeholder directory or delete state and lock files by hand.
If you cannot restore the original path, preserve the VM and data until you have
an appropriate recovery plan.

## Operation locks

The lock file's presence does not block VBX; a running process holding its kernel
lock does. If a command reports a busy instance, check for another active VBX
operation first. To inspect holders on the host (when `fuser` is installed):

```sh
name=$(vbx name)
lock="${XDG_STATE_HOME:-$HOME/.local/state}/vbx/locks/$name.lock"
fuser -v "$lock"
```

The launcher closes its lock descriptor for Lima commands and releases the lock
on exit. If a leftover Lima process holds a lock and no VBX operation is running,
stop only the affected instance directly:

```sh
name=$(vbx name)
limactl stop "$name"
```

This ends the instance's Lima processes. Retry the intended operation.
If it remains busy, inspect the lock holders (for example with `fuser -v` if
available) before stopping any process. Never delete the lock file while a holder
may still be alive: that creates a different inode and can bypass mutual exclusion.

## Report an unresolved problem

Include the following in a bug report:

- The exact command, expected behavior, actual result, and steps to reproduce.
- Whether this happened during first creation, a retry, or reuse of a ready VM.
- The installed source revision, if recorded. `git rev-parse HEAD` in the source
  checkout identifies that checkout's current revision, which may differ from
  the installed executable. VBX has no version flag.
- Host distribution and architecture, Lima/QEMU versions, and prerequisite results.
  Run these on the host:

  ```sh
  uname -sm
  cat /etc/os-release
  limactl --version
  qemu-system-x86_64 --version
  qemu-img --version
  vbx check-host-prerequisites
  ```

- From the affected project, include `vbx status`, the relevant `vbx list-vms`
  row, and sanitized terminal error output. For configuration problems, include
  the relevant parts of `vbx show-project-effective-setup`.

Review every attachment before sharing. Redact API keys, tokens, private paths,
and repository details. The effective-setup report hides managed environment
values, but filenames and other configuration can still be sensitive. Do not
attach `.env` files, private keys, or captured setup archives. Preserve failed
VMs until you have backed up valuable files and collected the needed diagnostics.
