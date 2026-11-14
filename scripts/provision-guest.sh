#!/usr/bin/env bash
# Runs as the guest user. System installers receive privilege explicitly.
set -euo pipefail
bundle=$1
project=$2
instance=$3
state="$HOME/.local/state/vbx/provision/$instance"
mkdir -p "$state"
chmod 700 "$state"
[[ ! -f "$state/complete" ]] || exit 0
export PATH="$HOME/.local/bin:$PATH"
# Inherited by npm, Pi's npm subprocesses, and project setup. These are policy
# settings, not an execution sandbox for explicitly approved setup code.
strict_npm() {
    unset NPM_CONFIG_STRICT_ALLOW_SCRIPTS NPM_CONFIG_IGNORE_SCRIPTS NPM_CONFIG_DANGEROUSLY_ALLOW_ALL_SCRIPTS
    unset npm_config_allow_scripts NPM_CONFIG_ALLOW_SCRIPTS
    export npm_config_strict_allow_scripts=true
    export npm_config_ignore_scripts=false
    export npm_config_dangerously_allow_all_scripts=false
}
strict_npm
if [[ -f "$bundle/apt.txt" && ! -f "$state/apt.done" ]]; then
    printf 'Installing declared APT requirements\n'
    sudo -n bash "$bundle/apt-install.sh" "$bundle/apt.txt"
    touch "$state/apt.done"
fi

while IFS= read -r module; do
    [[ -n "$module" ]] || continue
    [[ "$module" =~ ^[a-z][a-z0-9-]*$ ]] || { printf 'Invalid module identifier\n' >&2; exit 1; }
    [[ ! -f "$state/$module.done" ]] || continue
    printf 'Installing module: %s\n' "$module"
    bash "$bundle/modules/$module/setup.sh"
    touch "$state/$module.done"
done < "$bundle/modules.list"

# Also cover npm supplied explicitly through apt.packages, even without a recipe.
if [[ -f "$bundle/npm-policy.sh" ]] && command -v npm >/dev/null; then
    source "$bundle/npm-policy.sh"
    vbx_npm_policy
fi

for source in "$bundle/local-home" "$bundle/project-home"; do
    # Make conflicting directories writable while merging. cp restores each
    # source directory's mode afterward. HOME itself is never part of the copy.
    while IFS= read -r -d '' directory; do
        destination="$HOME/${directory#"$source/"}"
        if [[ -d "$destination" && ! -L "$destination" ]]; then
            chmod u+rwx -- "$destination"
        fi
    done < <(find "$source" -mindepth 1 -type d -print0)
    find "$source" -mindepth 1 -maxdepth 1 \
        -exec cp -a --remove-destination --no-preserve=ownership --target-directory="$HOME" -- {} +
done

# Keep the runtime environment separate from arbitrary user home overlays.
mkdir -p "$HOME/.config/vbx"
install -m 600 "$bundle/environment.sh" "$HOME/.config/vbx/environment.sh"
sudo -n bash "$bundle/guest-shell-system.sh"
source "$HOME/.config/vbx/environment.sh"
strict_npm

if [[ -f "$bundle/project-setup.sh" ]]; then
    printf 'Running project setup\n'
    cd -- "$project"
    bash "$bundle/project-setup.sh"
fi

touch "$state/complete"
printf 'Provisioning complete\n'
