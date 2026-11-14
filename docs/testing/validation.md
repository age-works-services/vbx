# Release validation

## Evidence

- The automated suites pass 66 checks.
- The operator reports that the current workflow works on a Linux host after the
  module-ownership and lock/failure-handling changes.
- The host report is an end-to-end confirmation, without individually recorded
  output for every procedure below.
- Automated tests use test doubles for Lima, APT, privilege escalation, and VS Code.
  They do not establish real QEMU, KVM, browser, editor, or API compatibility.
- Actual npm policy fixtures ran with Node 24.22.2 and npm 11.19.1 in the
  development container. Registry dry runs are separate from runtime validation.

## Run automated tests

From the checkout:

```sh
bash test/test-host.sh
bash test/test-vbx.sh
bash test/test-provision.sh
bash test/test-recipes.sh
bash test/test-pi-modules.sh
bash test/test-module-boundaries.sh
```

| Suite | Checks | Coverage |
| --- | ---: | --- |
| Host helper | 9 | Platform/tool/access checks, project input validation, canonical identity, VM name compatibility |
| Launcher | 28 | Configuration, lifecycle, approval, SSH/exec, locking, frozen inputs, failure guidance |
| Provisioning | 11 | Ordering, overlays, modes/ownership, environment, bootstrap, base upgrades |
| Recipe/npm | 11 | APT constraints, solver errors, npm version/policy enforcement, lifecycle failures |
| Pi modules | 4 | Minimal Pi, web-fetch ownership, missing prerequisites, failure propagation |
| Module boundaries | 3 | Local installer ownership, isolated Node module, explicit prerequisites |

The launcher requires no host Node/npm installation. Recipe tests need Node and
npm >=11.19 for the real lifecycle fixtures; they report a skip if either tool is
absent. Pi module fixtures also report a skip without Node. A full 66-check run
includes those fixtures.

APT and sudo fixtures never modify host packages or host `/etc`. npm fixtures
use synthetic local tarballs and temporary HOME/cache/prefix directories.
Production registry packages are not installed by these tests.

## Repeat the Linux host walkthrough

Use a disposable project and follow the [README](../../README.md).
Record host distribution, kernel, Lima/QEMU versions, selected modules, and command
output when collecting a new validation record.

1. Run `vbx --check`, create the VM, and confirm the account, OS, and workspace.
2. Open VS Code and a terminal. Verify project writes from both host and guest.
   Test additional writable/read-only mounts if configured.
3. Verify local/project home precedence, file modes, harmless environment values,
   and project setup's working directory. Do not print credentials.
4. Stop/start and reuse the VM. Confirm guest-home persistence and no repeated
   installation. Open another session while the first remains connected.
5. In a separate disposable project, force setup failure. Confirm no tool launch,
   an available operation lock afterward, and explicit remove/recreate guidance.
   Correct the inputs, remove the VM, and create it again.
6. Confirm removal asks for confirmation and leaves host project files intact.

For a Pi-equipped VM:

```sh
vbx --exec id -un
vbx --exec printenv HOME
vbx --exec node --version
vbx --exec npm --version
vbx --exec npm config get strict-allow-scripts
vbx --exec pi --version
vbx --exec pi list
vbx --exec pi mcp list
vbx --vscode
```

Expected account/home: `dev` and `/home/dev`. The Node/npm toolchain must meet the
module requirements, and strict npm policy reports `true`. Pi lists subagents and
prompt templates; web-fetch appears only when selected or separately configured.

With `pi-web-fetch` selected, verify the creation-time sandboxed rendering check
and an explicit Pi `web_fetch` call, including Python extraction. With Perplexity
configured, verify a harmless authenticated query. Package registration and npm
audit results do not substitute for these functional checks.

## Storage observations

Measure allocated storage from the project directory:

```sh
du -sh "${LIMA_HOME:-$HOME/.lima}/$(vbx --print-name)"
```

The operator measured approximately 3 GiB for an Ubuntu 26.04 Pi-equipped VM with
the earlier full development/browser bundle. That configuration predates the
module split, so it is a reference measurement rather than a first-release size
guarantee. Fifteen similarly sized instances would allocate approximately 45 GiB,
plus shared image cache and application growth. Browser/Python first-use caches
were not measured separately. The default 30 GiB disk setting is capacity, not
immediate allocation.

## Package review observations

A recorded registry preflight passed with exact script approvals for
`@google/genai@2.21.0`, `protobufjs@7.6.6`, and `puppeteer@24.43.1`. These are
script-policy observations, not a lockfile for the whole dependency tree.

A fresh Puppeteer 24.43.1 audit reported 10 high-severity dependency entries
(including inherited findings), with zero critical entries in that resolution.
The result excludes native Chrome vulnerabilities and does not describe every
existing guest lockfile. Upstream remediation is outside VBX's scope; see
[security notes](../manuals/recipes.md#security-notes).
