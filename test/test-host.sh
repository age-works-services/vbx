#!/usr/bin/env bash
# Offline host-helper tests. Never start VMs or modify host configuration.
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
HELPER="$ROOT/scripts/host.sh"
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
passed=0
pass() { passed=$((passed + 1)); printf 'PASS: %s\n' "$*"; }

bash -n "$HELPER"
TEMPLATE=unchanged
source "$HELPER"
[[ "$TEMPLATE" == unchanged ]]
! declare -F start_vm >/dev/null
! declare -F smoke >/dev/null
pass 'host library sources cleanly without template or legacy lifecycle side effects'

mkdir -p "$TMP/one/app" "$TMP/two/app"
set_project "$TMP/one/app"; first=$NAME
set_project "$TMP/two/app"; [[ "$NAME" != "$first" ]]
set_project "$TMP/one/app"; [[ "$NAME" == "$first" ]]
ln -s "$TMP/one/app" "$TMP/link"
set_project "$TMP/link"; [[ "$NAME" == "$first" ]]
pass 'canonical identity is stable and distinguishes equal basenames'

digest=$(printf '%s' "$(cd -- "$TMP/one/app" && pwd -P)" | sha256sum)
[[ "$first" == "vbx-app-${digest:0:12}" ]]
pass 'VM identity retains the existing production slug and twelve-character hash format'

mkdir -p "$TMP/"$'line\nbreak'
for invalid in / "$HELPER" "$TMP/missing" "$TMP/"$'line\nbreak'; do
    if (set_project "$invalid") > /dev/null 2>&1; then exit 1; fi
done
pass 'root, files, nonexistent directories, and line breaks are rejected'

mkdir -p "$TMP/ABCDEFGHIJKLMNOPQRSTUVWXYZ" "$TMP/!!!"
set_project "$TMP/ABCDEFGHIJKLMNOPQRSTUVWXYZ"
[[ "$NAME" == vbx-abcdefghijklmnopqrstuvwx-???????????? ]]
set_project "$TMP/!!!"
[[ "$NAME" == vbx-project-???????????? ]]
pass 'slugs normalize case, truncate predictably, and handle punctuation-only names'

if (
    uname() { printf 'Darwin\n'; }
    check_host --quiet
) > "$TMP/os-error" 2>&1; then exit 1; fi
grep -q 'Linux x86_64' "$TMP/os-error"
pass 'unsupported host operating systems fail before probing tools'

if (
    uname() { case "$1" in -s) printf 'Linux\n';; -m) printf 'aarch64\n';; esac; }
    check_host --quiet
) > "$TMP/arch-error" 2>&1; then exit 1; fi
grep -q 'Linux x86_64' "$TMP/arch-error"
pass 'unsupported host architectures fail before probing tools'

if (
    uname() { case "$1" in -s) printf 'Linux\n';; -m) printf 'x86_64\n';; esac; }
    command() { [[ "$1" == -v && "$2" != qemu-img ]]; }
    check_host --quiet
) > "$TMP/tools-error" 2>&1; then exit 1; fi
grep -q 'Missing host tools: qemu-img' "$TMP/tools-error"
grep -Fq "$ROOT/docs/manuals/host-setup.md" "$TMP/tools-error"
pass 'missing tools identify the command and current host setup manual'

(
    uname() { case "$1" in -s) printf 'Linux\n';; -m) printf 'x86_64\n';; esac; }
    command() { [[ "$1" == -v ]]; }
    run() { printf '%s\n' "$*" >> "$TMP/versions"; }
    if [[ -r /dev/kvm && -w /dev/kvm && $EUID != 0 ]]; then
        check_host --quiet > "$TMP/quiet"
        [[ ! -s "$TMP/quiet" && ! -e "$TMP/versions" ]]
        check_host > "$TMP/host-output"
        [[ $(wc -l < "$TMP/versions") == 4 ]]
        grep -q 'Host prerequisites passed' "$TMP/host-output"
    else
        if (check_host --quiet) > "$TMP/access-error" 2>&1; then exit 1; fi
        if [[ ! -r /dev/kvm || ! -w /dev/kvm ]]; then
            grep -q 'read/write access to /dev/kvm' "$TMP/access-error"
        else
            grep -q 'normal host user' "$TMP/access-error"
        fi
        [[ ! -e "$TMP/versions" ]]
    fi
)
pass 'host access checks gate version probes and quiet mode suppresses them'

printf '%s host-helper offline checks passed.\n' "$passed"
