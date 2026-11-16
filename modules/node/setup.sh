#!/usr/bin/env bash
set -euo pipefail
here=$(cd -- "$(dirname -- "$0")" && pwd -P)
# Reuse a compatible toolchain; a selected recipe does not require another recipe.
if command -v node >/dev/null && command -v npm >/dev/null &&
   node -e 'const [a,b,c]=process.versions.node.split(".").map(Number);process.exit(a>22 || (a===22 && (b>19 || (b===19 && c>=0))) ? 0 : 1)' &&
   npm --version | awk -F. '{exit !(($1>11)||($1==11&&$2>=19))}'; then
    :
else
    # Nonstandard package source bootstrap stays explicit in this installer.
    sudo -n bash "$here/system.sh"
fi
source "$here/npm-policy.sh"
vbx_npm_policy
