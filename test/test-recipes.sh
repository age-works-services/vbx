#!/usr/bin/env bash
# Offline policy tests. APT is mocked; npm installs only synthetic local fixtures.
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
T=$(mktemp -d)
trap 'chmod -R u+rwX -- "$T"; rm -rf -- "$T"' EXIT
mkdir -p "$T/project" "$T/bin" "$T/home"
export TEST_ROOT="$T" HOME="$T/home"
passed=0
pass() { passed=$((passed + 1)); printf 'PASS: %s\n' "$*"; }

printf 'version: 1\napt:\n  packages:\n    - git\n    - "nodejs (>= 22.19)"\nmodules:\n  - pi\n  - node\n' > "$T/local.yaml"
printf 'version: 1\napt:\n  packages:\n    - git\n    - "nodejs (<= 30)"\nmodules:\n  - pi\n  - pi-web-fetch\n' > "$T/project.yaml"
perl "$ROOT/scripts/config.pl" "$T/project" /workspaces/project "$T/local.yaml" "$T/project.yaml" > "$T/config"
apt=(); modules=()
while IFS= read -r -d '' key; do
    IFS= read -r -d '' value
    case "$key" in apt) apt+=("$value");; module) modules+=("$value");; esac
done < "$T/config"
[[ "${apt[*]}" == 'git nodejs (>= 22.19) nodejs (<= 30)' ]]
[[ "${modules[*]}" == 'pi node pi-web-fetch' ]]
pass 'APT layers deduplicate identical clauses, preserve constraints, and keep explicit recipe order'

printf 'version: 1\nmodules:\n  - browser\n' > "$T/removed.yaml"
if perl "$ROOT/scripts/config.pl" "$T/project" /workspaces/project "$T/removed.yaml" "$T/project.yaml" > "$T/removed-error" 2>&1; then exit 1; fi
grep -qi 'unknown.*browser' "$T/removed-error"
pass 'removed browser module is rejected in new configuration'

for invalid in 'https://example.invalid/pkg.deb' '--allow-unauthenticated' 'git;touch /tmp/pwned' 'git | bash' 'git (= latest)' 'git, sudo' 'Conflicts: git'; do
    printf 'version: 1\napt:\n  packages:\n    - "%s"\n' "$invalid" > "$T/bad.yaml"
    if perl "$ROOT/scripts/config.pl" "$T/project" /workspaces/project "$T/bad.yaml" "$T/project.yaml" > /dev/null 2>&1; then exit 1; fi
done
printf 'git\ngit\nnodejs (>= 22.19)\nnodejs (<= 30)\n' > "$T/input"
perl "$ROOT/scripts/apt-deps.pl" "$T/input" > "$T/apt.txt"
[[ $(< "$T/apt.txt") == $'git\nnodejs (>= 22.19)\nnodejs (<= 30)' ]]
printf 'git\n--allow-unauthenticated\n' > "$T/input"
if perl "$ROOT/scripts/apt-deps.pl" "$T/input" > /dev/null 2>&1; then exit 1; fi
pass 'both configuration and recipe APT manifests reject flags, shell expressions, alternatives and malformed versions'

cat > "$T/bin/apt-get" <<'SH'
#!/usr/bin/env bash
set -eu
printf '%s\n' "$*" >> "$TEST_ROOT/apt-log"
if [[ "$1" == satisfy ]]; then
    printf '%s\0' "$@" > "$TEST_ROOT/apt-args"
    [[ ${FAIL_APT:-0} == 0 ]] || exit 1
fi
SH
chmod +x "$T/bin/apt-get"
# Redirect the installer's only filesystem cleanup into the fixture.
installer=$(< "$ROOT/scripts/apt-install.sh")
installer=${installer//\/var\/lib\/apt\/lists/"$T/lists"}
[[ "$installer" != *'/var/lib/'* ]]
printf '%s\n' "$installer" > "$T/apt-install.sh"
PATH="$T/bin:$PATH" bash "$T/apt-install.sh" "$T/apt.txt"
mapfile -d '' -t args < "$T/apt-args"
[[ ${args[0]} == satisfy && ${args[2]} == --no-remove && ${args[4]} == -- ]]
[[ ${args[6]} == 'nodejs (>= 22.19)' && ${args[7]} == 'nodejs (<= 30)' ]]
if FAIL_APT=1 PATH="$T/bin:$PATH" bash "$T/apt-install.sh" "$T/apt.txt" > /dev/null 2>&1; then exit 1; fi
pass 'APT receives literal native constraints with no removals, and solver failures propagate'

# Guest APT failure must block every recipe and retain an incomplete state.
mkdir -p "$T/bundle/"{local-home,project-home,modules/check}
cp "$T/apt-install.sh" "$T/bundle/apt-install.sh"
cp "$T/apt.txt" "$T/bundle/apt.txt"
cp "$ROOT/scripts/guest-shell-system.sh" "$T/bundle/"
printf 'check\n' > "$T/bundle/modules.list"
printf 'export NPM_CONFIG_DANGEROUSLY_ALLOW_ALL_SCRIPTS=true\nexport npm_config_strict_allow_scripts=false\n' > "$T/bundle/environment.sh"
cat > "$T/bundle/project-setup.sh" <<'SH'
set -eu
[[ "${npm_config_strict_allow_scripts}" == true ]]
[[ "${npm_config_dangerously_allow_all_scripts}" == false ]]
[[ -z "${NPM_CONFIG_DANGEROUSLY_ALLOW_ALL_SCRIPTS:-}" ]]
SH
printf 'touch "$TEST_ROOT/recipe-ran"\n' > "$T/bundle/modules/check/setup.sh"
cat > "$T/bin/sudo" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == -n ]]; shift
if [[ "$2" == */guest-shell-system.sh ]]; then exit 0; fi
exec "$@"
SH
chmod +x "$T/bin/sudo"
guest() { PATH="$T/bin:$PATH" bash "$ROOT/scripts/provision-guest.sh" "$T/bundle" "$T/project" apt-test; }
if FAIL_APT=1 guest > "$T/guest-log" 2>&1; then exit 1; fi
state="$HOME/.local/state/vbx/provision/apt-test"
[[ ! -e "$T/recipe-ran" && ! -f "$state/apt.done" && ! -f "$state/complete" ]]
guest > "$T/guest-log"
[[ -f "$T/recipe-ran" && -f "$state/apt.done" && -f "$state/complete" ]]
cp "$T/apt-log" "$T/previous-apt"
guest > /dev/null
cmp "$T/apt-log" "$T/previous-apt"
pass 'failed APT blocks recipe/launch readiness; retry completes and ordinary reuse skips setup'

# Guard old npm without running any installer or mutating real configuration.
cat > "$T/bin/npm" <<'SH'
#!/usr/bin/env bash
[[ "$1" == --version ]] || exit 99
printf '11.18.0\n'
SH
chmod +x "$T/bin/npm"
if (export PATH="$T/bin:$PATH"; source "$ROOT/modules/node/npm-policy.sh"; vbx_npm_policy) > "$T/old-npm" 2>&1; then exit 1; fi
grep -q 'npm >=11.19.0 required' "$T/old-npm"
rm "$T/bin/npm"
pass 'unsupported npm stops before a silently ignored approval policy can be used'

if ! command -v node >/dev/null || ! command -v npm >/dev/null; then
    printf 'SKIP: real npm fixture checks require Node and npm >=11.19 (launcher does not).\n'
    printf '%s recipe checks passed.\n' "$passed"
    exit 0
fi

# Package code is entirely synthetic; no registry or production packages are used.
export npm_config_cache="$T/npm-cache" npm_config_update_notifier=false npm_config_audit=false npm_config_fund=false
mkdir -p "$T/fixture" "$HOME/.pi/agent/npm"
printf '%s\n' '{"name":"vbx-script-fixture","version":"1.0.0","scripts":{"postinstall":"node install.cjs"}}' > "$T/fixture/package.json"
printf 'require("node:fs").writeFileSync(process.env.HOME + "/script-ran", "ok");\n' > "$T/fixture/install.cjs"
npm pack --ignore-scripts --pack-destination "$T" "$T/fixture" > /dev/null 2>&1
export TEST_TARBALL="$T/vbx-script-fixture-1.0.0.tgz" TEST_APP="$ROOT"
cat > "$T/bin/pi" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == install ]]
exec npm install --offline --prefix "$HOME/.pi/agent/npm" "$TEST_TARBALL"
SH
chmod +x "$T/bin/pi"
printf '%s\n' '{"private":true}' > "$HOME/.pi/agent/npm/package.json"
cat > "$T/bundle/modules/check/setup.sh" <<'SH'
set -euo pipefail
source "$TEST_APP/modules/node/npm-policy.sh"
vbx_npm_policy
pi install npm:vbx-script-fixture
SH
npm_guest() { PATH="$T/bin:$PATH" bash "$ROOT/scripts/provision-guest.sh" "$T/bundle" "$T/project" npm-test; }
# Pre-mark the already-tested APT phase so only the npm subprocess path is exercised.
state="$HOME/.local/state/vbx/provision/npm-test"
mkdir -p "$state"; touch "$state/apt.done"
if npm_guest > "$T/npm-error" 2>&1; then exit 1; fi
grep -q ESTRICTALLOWSCRIPTS "$T/npm-error"
[[ ! -f "$HOME/script-ran" && ! -f "$state/check.done" && ! -f "$state/complete" ]]
pass 'real npm blocks unreviewed scripts through a Pi-like subprocess and guest readiness remains incomplete'

node -e 'require("node:fs").writeFileSync(process.argv[1], JSON.stringify({private:true,allowScripts:{["file:" + process.env.TEST_TARBALL]:true}}))' "$HOME/.pi/agent/npm/package.json"
npm_guest > "$T/npm-output" 2>&1 || { tail -n 40 "$T/npm-output"; exit 1; }
[[ $(< "$HOME/script-ran") == ok && -f "$state/check.done" && -f "$state/complete" ]]
pass 'an explicit source-scoped approval runs the script and allows provisioning to complete'

rm -rf "$HOME/.pi/agent/npm/node_modules" "$HOME/.pi/agent/npm/package-lock.json"
rm "$HOME/script-ran"
printf '%s\n' '{"private":true,"allowScripts":{"vbx-script-fixture@0.9.0":true}}' > "$HOME/.pi/agent/npm/package.json"
if PATH="$T/bin:$PATH" pi install npm:fixture > "$T/wrong-version" 2>&1; then exit 1; fi
grep -q ESTRICTALLOWSCRIPTS "$T/wrong-version"
[[ ! -f "$HOME/script-ran" ]]
pass 'a registry name/version approval cannot authorize a local tarball source'

# A reviewed script's own failure must propagate just like a policy rejection.
node - "$T/fixture/package.json" <<'NODE'
const fs = require('node:fs');
const file = process.argv[2];
const data = JSON.parse(fs.readFileSync(file)); data.version = '1.1.0';
fs.writeFileSync(file, JSON.stringify(data));
NODE
printf 'process.exit(7);\n' > "$T/fixture/install.cjs"
npm pack --ignore-scripts --pack-destination "$T" "$T/fixture" > /dev/null 2>&1
export TEST_TARBALL="$T/vbx-script-fixture-1.1.0.tgz"
node -e 'require("node:fs").writeFileSync(process.argv[1], JSON.stringify({private:true,allowScripts:{["file:" + process.env.TEST_TARBALL]:true}}))' "$HOME/.pi/agent/npm/package.json"
state="$HOME/.local/state/vbx/provision/npm-script-failure"
mkdir -p "$state"; touch "$state/apt.done"
if PATH="$T/bin:$PATH" bash "$ROOT/scripts/provision-guest.sh" "$T/bundle" "$T/project" npm-script-failure > "$T/script-error" 2>&1; then exit 1; fi
grep -q 'code 7' "$T/script-error"
[[ ! -f "$state/check.done" && ! -f "$state/complete" ]]
pass 'reviewed lifecycle scripts that fail also block provisioning completion'

node - "$ROOT" <<'NODE'
const assert = require('node:assert/strict');
const root = process.argv[2];
assert.deepEqual(require(`${root}/modules/pi/core-npm-policy.json`).allowScripts, {
    '@google/genai@2.21.0': true, 'protobufjs@7.6.6': true,
});
assert.deepEqual(require(`${root}/modules/pi-web-fetch/npm-policy.json`).allowScripts, {'puppeteer@24.43.1':true});
assert.equal(require('node:fs').existsSync(`${root}/modules/browser`), false);
NODE
pass 'Pi and web-fetch approvals remain exact versions without a standalone browser module'
printf '%s recipe checks passed.\n' "$passed"
