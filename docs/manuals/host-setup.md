# Missing host prerequisites

Run only the section matching the failed check, then rerun:

```sh
vbx check-host-prerequisites
```

## `limactl` missing

Requires `curl` and `tar`. On Ubuntu/Debian, install them if missing:

```sh
sudo apt-get --no-remove install curl tar
```

Install the latest stable Lima bundle in `$HOME/.local/lib/lima` and link its
commands into `$HOME/.local/bin`:

```sh
export PATH="$HOME/.local/bin:$PATH"
prefix="$HOME/.local/lib/lima"
archive="$(mktemp)"
release_url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' https://github.com/lima-vm/lima/releases/latest)" &&
    tag="${release_url##*/}" &&
    curl -fL "https://github.com/lima-vm/lima/releases/download/$tag/lima-${tag#v}-Linux-x86_64.tar.gz" -o "$archive" &&
    mkdir -p "$prefix" "$HOME/.local/bin" &&
    tar -xzf "$archive" -C "$prefix" &&
    rm -f "$archive" &&
    ln -s "$prefix/bin/limactl" "$HOME/.local/bin/limactl" &&
    ln -s "$prefix/bin/lima" "$HOME/.local/bin/lima" &&
    limactl --version
```

Add `$HOME/.local/bin` to your shell's persistent PATH.
Keep the bundle's directory layout intact so Lima can locate its supporting files.

## `qemu-system-x86_64` or `qemu-img` missing

First check whether the existing binaries are outside your PATH:

```sh
ls -l /usr/bin/qemu-system-x86_64 /usr/bin/qemu-img
```

If both exist, add `/usr/bin` to PATH:

```sh
export PATH="/usr/bin:$PATH"
```

Otherwise, on Ubuntu/Debian, check for the HWE package:

```sh
dpkg-query -W -f='${db:Status-Abbrev}\n' qemu-system-x86-hwe
```

If it reports `ii`, use the HWE packages:

```sh
sudo apt-get --no-remove install qemu-system-x86-hwe qemu-utils-hwe
```

If it reports no installed HWE package, use the standard packages:

```sh
sudo apt-get --no-remove install qemu-system-x86 qemu-utils
```

`--no-remove` aborts if dependency resolution requires removing packages. If it
aborts, retain the existing stack and resolve the package conflict before continuing.
On other distributions, use the package names supplied by your distribution.

## `ssh` missing

On Ubuntu/Debian:

```sh
sudo apt-get --no-remove install openssh-client
```

## `/dev/kvm` access missing

Check the device and current groups:

```sh
ls -l /dev/kvm
id
```

If the device belongs to group `kvm` and your user is not a member:

```sh
sudo usermod -aG kvm "$USER"
```

Sign out and back in, then rerun the check. If `/dev/kvm` does not exist, enable
hardware virtualization in the host firmware, or nested virtualization in the
outer hypervisor when the host is itself a VM.

## References

- [Lima installation](https://lima-vm.io/docs/installation/)
- [Latest stable Lima release](https://github.com/lima-vm/lima/releases/latest)
