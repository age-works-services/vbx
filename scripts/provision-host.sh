#!/usr/bin/env bash
# Sourced by vbx. Snapshot approved setup before starting the guest.
CLEANUP_DIRS=()

cleanup_bundles() {
    local directory
    for directory in "${CLEANUP_DIRS[@]}"; do
        # Overlays may contain read-only directories. These are private copies.
        chmod -R u+rwX -- "$directory" || return
        rm -rf -- "$directory" || return
    done
    CLEANUP_DIRS=()
}

prepare_bundle() {
    local bundle module source layer
    local apt_files=()
    bundle=$(mktemp -d)
    CLEANUP_DIRS+=("$bundle")
    mkdir -p "$bundle/modules" "$bundle/local-home" "$bundle/project-home"
    cp "$VBX_ROOT/scripts/provision-guest.sh" "$bundle/provision.sh"
    cp "$VBX_ROOT/scripts/guest-shell-system.sh" "$bundle/guest-shell-system.sh"
    cp "$VBX_ROOT/scripts/apt-install.sh" "$bundle/apt-install.sh"
    cp -- "$VBX_ROOT/scripts/npm-policy.sh" "$bundle/npm-policy.sh"
    printf '%s\n' "${APT_PACKAGES[@]}" > "$bundle/extra-apt.txt"
    apt_files+=("$bundle/extra-apt.txt")
    : > "$bundle/modules.list"
    for module in "${MODULES[@]}"; do
        cp -a -- "$VBX_ROOT/modules/$module" "$bundle/modules/"
        [[ -f "$bundle/modules/$module/setup.sh" ]] || fail "Missing recipe setup: $module"
        if [[ -f "$bundle/modules/$module/apt.txt" ]]; then
            apt_files+=("$bundle/modules/$module/apt.txt")
        fi
        printf '%s\n' "$module" >> "$bundle/modules.list"
    done
    perl "$VBX_ROOT/scripts/apt-deps.pl" "${apt_files[@]}" > "$bundle/apt.txt"
    for layer in local project; do
        if [[ "$layer" == local ]]; then source="$VBX_ROOT/config.local/home"
        else source="$PROJECT/.vbx/home"; fi
        if [[ -e "$source" || -L "$source" ]]; then
            [[ -d "$source" ]] || fail "Expected a home directory: $source"
            cp -a --no-preserve=ownership -- "$source/." "$bundle/$layer-home/"
        fi
    done
    source="$PROJECT/.vbx/setup.sh"
    if [[ -e "$source" || -L "$source" ]]; then
        [[ -f "$source" ]] || fail 'Project setup.sh must be a regular file.'
        cp -L -- "$source" "$bundle/project-setup.sh"
    fi
    perl "$VBX_ROOT/scripts/environment.pl" "${ENV_FILES[@]}" > "$bundle/environment.sh"
    chmod 600 "$bundle/environment.sh"
    # State is private; never put the archive or environment values in workspace mounts.
    tar -C "$bundle" -cf "$STATE/provision.tar" .
    chmod 600 "$STATE/provision.tar"
}

provision_vm() {
    run limactl shell "--workdir=$PRIMARY" "$NAME" -- bash -eu -c '
        bundle=$(mktemp -d)
        trap '\''chmod -R u+rwX -- "$bundle"; rm -rf -- "$bundle"'\'' EXIT
        tar -xf - -C "$bundle"
        bash "$bundle/provision.sh" "$bundle" "$1" "$2"
    ' vbx-provision "$PRIMARY" "$NAME" < "$STATE/provision.tar"
}
