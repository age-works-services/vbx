#!/usr/bin/env bash
# Sourced host checks and canonical project identity. Version output uses run().

fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

set_project() {
    [[ -d "$1" ]] || fail "Project directory does not exist: $1"
    PROJECT=$(cd -- "$1" && pwd -P)
    [[ "$PROJECT" != / ]] || fail 'Select a project directory other than /.'
    case "$PROJECT" in
        *$'\n'*|*$'\r'*) fail 'Project paths containing line breaks are unsupported.' ;;
    esac
    local slug digest
    slug=$(basename -- "$PROJECT" | LC_ALL=C tr '[:upper:]' '[:lower:]' |
        LC_ALL=C sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')
    slug=${slug:0:24}
    digest=$(printf '%s' "$PROJECT" | sha256sum)
    NAME="vbx-${slug:-project}-${digest:0:12}"
}

check_host() {
    [[ $(uname -s) == Linux && $(uname -m) == x86_64 ]] ||
        fail 'VBX supports Linux x86_64 hosts.'
    local tool root missing=()
    root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
    for tool in limactl qemu-system-x86_64 qemu-img ssh sha256sum mktemp sed tr; do
        command -v "$tool" >/dev/null || missing+=("$tool")
    done
    [[ ${#missing[@]} == 0 ]] || fail "Missing host tools: ${missing[*]}. Setup: $root/docs/manuals/host-setup.md"
    [[ -r /dev/kvm && -w /dev/kvm ]] || fail "The host user needs read/write access to /dev/kvm. Setup: $root/docs/manuals/host-setup.md"
    [[ $EUID != 0 ]] || fail 'Run as your normal host user, without sudo.'
    if [[ ${1:-} != --quiet ]]; then
        run limactl --version
        run qemu-system-x86_64 --version
        run qemu-img --version
        run ssh -V
        printf 'Host prerequisites passed.\n'
    fi
}

