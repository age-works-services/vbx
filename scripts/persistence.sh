#!/usr/bin/env bash
# Apply and verify captured, validated directory mappings as the guest user.
set -euo pipefail
mode=$1
records=$2
instance=$3
fail() {
    printf 'Persistence error: %s\nGuest: %s\nRetained: %s\n' "$1" "${guest_path:-unknown}" "${target:-unknown}" >&2
    printf 'Inspect with: limactl shell --workdir=/ %q\nPreserve guest-local data before removing the VM.\n' "$instance" >&2
    exit 1
}
# Reject symlink ancestors even above the user-owned home or module mount.
real_chain() {
    local current= part
    local -a chain=()
    [[ "$1" == /* ]] || fail 'directory root is not absolute'
    IFS=/ read -r -a chain <<< "${1#/}"
    for part in "${chain[@]}"; do
        current="$current/$part"
        [[ -d "$current" && ! -L "$current" ]] || fail "invalid directory ancestor $current"
    done
}
# Check each existing component without following symlinks; optionally create it.
directories() {
    local current=$1 relative=$2 create=$3 part
    real_chain "$current"
    [[ -d "$current" && ! -L "$current" && -O "$current" ]] || fail "invalid directory $current"
    local -a parts=()
    IFS=/ read -r -a parts <<< "$relative"
    for part in "${parts[@]}"; do
        [[ -n "$part" && "$part" != . && "$part" != .. ]] || fail 'invalid path component'
        current="$current/$part"
        if [[ ! -e "$current" && ! -L "$current" && "$create" == yes ]]; then mkdir -m 700 -- "$current"; fi
        [[ -d "$current" && ! -L "$current" && -O "$current" ]] || fail "invalid directory $current"
    done
}
if [[ -f "$records/bindings.tsv" ]]; then
    while IFS=$'\t' read -r module root expected stable; do
        [[ -n "$module" ]] || continue
        guest_path=$root target=$root
        info=$(findmnt -rn -M "$root" -o FSTYPE,SOURCE,TARGET) || fail 'persistent mount missing'
        read -r type source actual extra <<< "$info"
        if [[ "$info" == *$'\n'* || "$type" != 9p || "$actual" != "$root" || -n "$extra" ]] ||
            [[ "$source" != "$expected" && ( -z "$stable" || "$source" != "$stable" ) ]]; then
            fail "persistent mount differs from captured binding (expected 9p source $expected${stable:+ or $stable} at $root; found $info)"
        fi
        directories "$root" '' no
        [[ -w "$root" ]] || fail 'persistent mount is not writable'
    done < "$records/bindings.tsv"
fi
[[ "$mode" != mounts ]] || exit 0
binding_root() {
    local name root tag stable
    while IFS=$'\t' read -r name root tag stable; do
        if [[ "$name" == "$1" ]]; then module_root=$root; return; fi
    done < "$records/bindings.tsv"
    fail 'mapping has no captured module binding'
}
[[ -f "$records/mappings.tsv" ]] || exit 0
while IFS=$'\t' read -r module relative data; do
    [[ -n "$module" ]] || continue
    binding_root "$module"
    guest_path="$HOME/${relative:2}"
    target="$module_root/$data"
    create=no; [[ "$mode" != apply ]] || create=yes
    directories "$module_root" "$data" "$create"
    parent=${relative:2}; parent=${parent%/*}
    if [[ "${relative:2}" != */* ]]; then parent=; fi
    directories "$HOME" "$parent" "$create"
    if [[ -L "$guest_path" ]]; then
        [[ $(readlink -- "$guest_path") == "$target" ]] || fail 'conflicting symbolic link'
    elif [[ "$mode" == apply ]]; then
        if [[ -e "$guest_path" ]]; then
            [[ -d "$guest_path" && -O "$guest_path" ]] || fail 'conflicting guest path'
            rmdir -- "$guest_path" 2>/dev/null || fail 'guest directory contains data; resolve both locations explicitly'
        fi
        ln -s -- "$target" "$guest_path" || fail 'could not install persistence link'
    else
        fail 'persistent connection missing or replaced'
    fi
done < "$records/mappings.tsv"
