#!/usr/bin/env bash
set -euo pipefail
here=$(cd -- "$(dirname -- "$0")" && pwd -P)
export PATH="$HOME/.local/bin:$PATH"
command -v node >/dev/null && command -v npm >/dev/null || {
    printf 'pi requires Node/npm; select node before pi.\n' >&2
    exit 1
}
source "$here/npm-policy.sh"
vbx_npm_policy
# Reviewed global-install scripts: Google's no-op preinstall and protobufjs's
# dependency-version warning. Exact versions prevent blanket future approval.
allowed=$(node -e '
    const policy = require(process.argv[1]).allowScripts;
    if (!policy || Object.values(policy).some(value => value !== true)) {
        throw new Error("Global npm policy must contain explicit positive approvals only");
    }
    console.log(Object.keys(policy).join(","));
' "$here/core-npm-policy.json")
npm install -g --strict-allow-scripts "--allow-scripts=$allowed" @earendil-works/pi-coding-agent
for package in pi-subagents pi-prompt-template-model; do
    pi install "npm:$package"
    pi list | grep -F "npm:$package" > /dev/null
done
pi --version
