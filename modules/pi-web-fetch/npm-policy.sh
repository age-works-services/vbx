#!/usr/bin/env bash
# Source in recipes before invoking npm. npm owns approval enforcement.
vbx_check_npm() {
    local version major minor
    version=$(npm --version) || return
    if [[ ! "$version" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
        printf 'Cannot validate npm version: %s\n' "$version" >&2; return 1
    fi
    major=${BASH_REMATCH[1]}; minor=${BASH_REMATCH[2]}
    # Require the lifecycle-policy contract tested with npm 11.19.1.
    if ((major < 11 || (major == 11 && minor < 19))); then
        printf 'npm >=11.19.0 required for strict script approvals; found %s\n' "$version" >&2
        return 1
    fi
}

vbx_npm_policy() {
    vbx_check_npm || return
    unset NPM_CONFIG_STRICT_ALLOW_SCRIPTS NPM_CONFIG_IGNORE_SCRIPTS NPM_CONFIG_DANGEROUSLY_ALLOW_ALL_SCRIPTS
    export npm_config_strict_allow_scripts=true
    export npm_config_ignore_scripts=false
    export npm_config_dangerously_allow_all_scripts=false
    # Policy must come from a managed manifest, never a caller's broad env allowlist.
    unset npm_config_allow_scripts NPM_CONFIG_ALLOW_SCRIPTS
    mkdir -p "$HOME/.local/bin" "$HOME/.local/lib/node_modules"
    npm config set prefix "$HOME/.local" --location=user
    npm config set strict-allow-scripts true --location=user
}
