#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
command -v node >/dev/null || { printf 'SKIP: module fixtures require Node.\n'; exit 0; }
T=$(mktemp -d)
trap 'rm -rf -- "$T"' EXIT
export HOME="$T/home" TEST_ROOT="$T" REAL_NODE
REAL_NODE=$(command -v node)
mkdir -p "$HOME" "$T/bin" "$T/bundle/modules"
cp -a "$ROOT/modules/pi" "$ROOT/modules/pi-web-fetch" "$T/bundle/modules/"
for module in pi pi-web-fetch; do
    printf 'vbx_npm_policy() { export npm_config_strict_allow_scripts=true; }\n' > "$T/bundle/modules/$module/npm-policy.sh"
done
: > "$T/log"
export PATH="$T/bin:/usr/bin:/bin"
if bash "$T/bundle/modules/pi-web-fetch/setup.sh" > "$T/missing" 2>&1; then exit 1; fi
grep -q 'select pi before pi-web-fetch' "$T/missing"
[[ ! -s "$T/log" ]]
printf 'PASS: web-fetch without Pi fails clearly before installer side effects\n'

cat > "$T/bin/npm" <<'SH'
#!/usr/bin/env bash
set -eu
[[ "$npm_config_strict_allow_scripts" == true ]]
printf 'npm %s\n' "$*" >> "$TEST_ROOT/log"
SH
cat > "$T/bin/pi" <<'SH'
#!/usr/bin/env bash
set -eu
printf 'pi %s\n' "$*" >> "$TEST_ROOT/log"
case "$1" in
    install)
        [[ "${FAIL_FETCH:-0}" != 1 || "$2" != npm:pi-web-fetch ]] || exit 1
        printf '%s\n' "$2" >> "$TEST_ROOT/packages";;
    list) sort -u "$TEST_ROOT/packages";;
    --version) printf 'test-pi\n';;
    *) exit 1;;
esac
SH
cat > "$T/bin/sudo" <<'SH'
#!/usr/bin/env bash
set -eu
[[ "$1" == -n && "$3" == */browser-system.sh ]]
printf 'browser-policy\n' >> "$TEST_ROOT/log"
SH
cat > "$T/bin/node" <<'SH'
#!/usr/bin/env bash
set -eu
if [[ "$1" == */check-browser.cjs ]]; then
    [[ "$2" == "$HOME/.pi/agent/npm/node_modules/pi-web-fetch/package.json" ]]
    printf 'web-fetch-render-check\n' >> "$TEST_ROOT/log"
    exit 0
fi
exec "$REAL_NODE" "$@"
SH
chmod +x "$T/bin/"*
bash "$T/bundle/modules/pi/setup.sh" > /dev/null
grep -q '^npm install -g .*strict-allow-scripts' "$T/log"
grep -Fxq npm:pi-subagents "$T/packages"
grep -Fxq npm:pi-prompt-template-model "$T/packages"
! grep -Eq 'browser|puppeteer|pi-web-fetch' "$T/log"
! grep -Eq '^(python3|python3-venv|pipx|libgtk-3-0t64|apparmor)$' "$ROOT/modules/pi/apt.txt"
[[ ! -e "$ROOT/config.local.example/home/.pi/agent/settings.json" ]]
printf 'PASS: Pi alone installs CLI and two extensions without browser/Python/web-fetch or settings auto-install\n'

mkdir -p "$HOME/.pi/agent/npm"
printf '%s\n' '{"dependencies":{"keep":"1"},"allowScripts":{"keep@1.0.0":false}}' > "$HOME/.pi/agent/npm/package.json"
: > "$T/log"
bash "$T/bundle/modules/pi-web-fetch/setup.sh"
grep -q '^browser-policy$' "$T/log"
grep -q '^pi install npm:pi-web-fetch$' "$T/log"
grep -q '^web-fetch-render-check$' "$T/log"
! grep -q '^npm ' "$T/log"
[[ ! -e "$HOME/.local/lib/vbx-browser" && ! -e "$HOME/.local/bin/vbx-browser" ]]
node - "$HOME/.pi/agent/npm/package.json" <<'NODE'
const assert = require('node:assert/strict');
const data = require(process.argv[2]);
assert.equal(data.dependencies.keep, '1');
assert.deepEqual(data.allowScripts, {'keep@1.0.0':false,'puppeteer@24.43.1':true});
NODE
for package in python3 python3-venv pipx apparmor; do
    grep -Fxq "$package" "$ROOT/modules/pi-web-fetch/apt.txt"
done
printf 'PASS: web-fetch owns prerequisites/policy/rendering, preserves other policies and avoids standalone browser installation\n'

: > "$T/log"
if FAIL_FETCH=1 bash "$T/bundle/modules/pi-web-fetch/setup.sh" > /dev/null 2>&1; then exit 1; fi
! grep -q '^web-fetch-render-check$' "$T/log"
printf 'PASS: extension installation failure propagates before rendering validation\n'
printf '4 Pi module checks passed.\n'
