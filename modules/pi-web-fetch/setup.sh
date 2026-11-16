#!/usr/bin/env bash
set -euo pipefail
here=$(cd -- "$(dirname -- "$0")" && pwd -P)
export PATH="$HOME/.local/bin:$PATH"
command -v pi >/dev/null || {
    printf 'pi-web-fetch requires Pi; select pi before pi-web-fetch.\n' >&2
    exit 1
}
source "$here/npm-policy.sh"
vbx_npm_policy
export PUPPETEER_SKIP_CHROME_HEADLESS_SHELL_DOWNLOAD=true
sudo -n bash "$here/browser-system.sh" "$HOME"
# Own only this extension's policy. Preserve other reviewed package approvals
# and dependencies on retry. npm resolves the extension's Puppeteer dependency.
mkdir -p "$HOME/.pi/agent/npm"
node - "$HOME/.pi/agent/npm/package.json" "$here/npm-policy.json" <<'NODE'
const fs = require("node:fs");
const [destination, policyPath] = process.argv.slice(2);
const data = fs.existsSync(destination) ? JSON.parse(fs.readFileSync(destination, "utf8")) : {private: true};
data.allowScripts = {...data.allowScripts, ...JSON.parse(fs.readFileSync(policyPath, "utf8")).allowScripts};
fs.writeFileSync(destination, JSON.stringify(data, null, 2) + "\n", {mode: 0o600});
NODE
pi install npm:pi-web-fetch
pi list | grep -F "npm:pi-web-fetch" > /dev/null
node "$here/check-browser.cjs" "$HOME/.pi/agent/npm/node_modules/pi-web-fetch/package.json"
