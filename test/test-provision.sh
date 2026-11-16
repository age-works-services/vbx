#!/usr/bin/env bash
# Offline guest execution with fake system installers. Never installs host packages.
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
T=$(mktemp -d)
trap 'chmod -R u+rwX -- "$T"; rm -rf -- "$T"' EXIT
export HOME="$T/home" TEST_ROOT="$T" TEST_LOG="$T/log"
mkdir -p "$HOME" "$T/bin" "$T/project" "$T/bundle/"{local-home,project-home,modules/alpha,modules/beta}
chmod 750 "$HOME"
export PATH="$T/bin:$PATH"
cat > "$T/bin/sudo" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == -n ]]; shift
if [[ "$2" == */guest-shell-system.sh ]]; then
    printf 'shell-profile\n' >> "$TEST_LOG"
else "$@"; fi
SH
chmod +x "$T/bin/sudo"
cp "$ROOT/scripts/guest-shell-system.sh" "$T/bundle/"
printf 'alpha\nbeta\n' > "$T/bundle/modules.list"
for module in alpha beta; do
    printf '#!/bin/bash\nset -eu\ntest -z "${SECRET:-}"\nprintf "system-%s\\n" >> "$TEST_LOG"\n' "$module" > "$T/bundle/modules/$module/system.sh"
    printf '#!/bin/bash\nset -eu\nprintf "user-%s\\n" >> "$TEST_LOG"\n' "$module" > "$T/bundle/modules/$module/user.sh"
    printf '#!/bin/bash\nset -eu\nhere=$(dirname -- "$0")\nsudo -n bash "$here/system.sh" "$HOME"\nbash "$here/user.sh"\n' > "$T/bundle/modules/$module/setup.sh"
done
cat >> "$T/bundle/modules/alpha/user.sh" <<'SH'
mkdir -p "$HOME/.pi/agent"
printf 'keep\n' > "$HOME/.pi/agent/module-only"
SH
cat >> "$T/bundle/modules/beta/user.sh" <<'SH'
test ! -f "$TEST_ROOT/fail-module"
SH
mkdir -p "$T/bundle/local-home/.pi/agent" "$T/bundle/project-home/.pi/agent"
printf 'local\n' > "$T/bundle/local-home/.pi/agent/setting"
printf 'project\n' > "$T/bundle/project-home/.pi/agent/setting"
printf 'dotfile\n' > "$T/bundle/local-home/.bashrc"
chmod 600 "$T/bundle/local-home/.bashrc"
chmod 750 "$T/bundle/local-home/.pi"
chmod 700 "$T/bundle/project-home/.pi"
for layer in local project; do
    mkdir "$T/bundle/$layer-home/.readonly"
    printf '%s\n' "$layer" > "$T/bundle/$layer-home/.readonly/setting"
    chmod 400 "$T/bundle/$layer-home/.readonly/setting"
    chmod 500 "$T/bundle/$layer-home/.readonly"
done
printf 'existing\n' > "$HOME/unrelated"
printf 'ORDER=local\nSECRET=literal $() `command` '\'' quote\n' > "$T/local.env"
printf 'ORDER=project\nEMPTY=\n' > "$T/project.env"
cat > "$T/bundle/environment.sh" <<'ENV'
export ORDER='project'
export EMPTY=''
export SECRET='literal $() `command` '\'' quote'
ENV
cat > "$T/bundle/project-setup.sh" <<'SH'
set -euo pipefail
[[ "$PWD" == "$TEST_ROOT/project" && "$ORDER" == project && -z "$EMPTY" ]]
[[ "$SECRET" == 'literal $() `command` '\'' quote' ]]
[[ $(< "$HOME/.pi/agent/setting") == project ]]
printf 'project-setup\n' >> "$TEST_LOG"
test ! -f "$TEST_ROOT/fail-project"
printf 'setup-owned\n' > "$HOME/.pi/agent/setting"
SH
run_guest() { bash "$ROOT/scripts/provision-guest.sh" "$T/bundle" "$T/project" instance; }
state="$HOME/.local/state/vbx/provision/instance"
passed=0
pass() { passed=$((passed + 1)); printf 'PASS: %s\n' "$*"; }

touch "$T/fail-module"
if run_guest > "$T/output" 2>&1; then exit 1; fi
[[ -f "$state/alpha.done" && ! -f "$state/beta.done" && ! -f "$state/complete" ]]
[[ ! -f "$HOME/.bashrc" ]]
pass 'module failure blocks overlays, environment, and project setup'

rm "$T/fail-module"
touch "$T/fail-project"
if run_guest > "$T/output" 2>&1; then exit 1; fi
[[ ! -f "$state/complete" && -f "$state/beta.done" ]]
[[ $(grep -c '^user-alpha$' "$TEST_LOG") == 1 && $(grep -c '^user-beta$' "$TEST_LOG") == 2 ]]
[[ $(< "$HOME/.pi/agent/setting") == project ]]
pass 'retry skips completed modules; failed project setup remains incomplete'

rm "$T/fail-project"
run_guest > "$T/output"
[[ -f "$state/complete" && $(< "$HOME/.pi/agent/setting") == setup-owned ]]
[[ $(grep -c '^user-beta$' "$TEST_LOG") == 2 ]]
[[ $(< "$TEST_LOG") == $'system-alpha\nuser-alpha\nsystem-beta\nuser-beta\nsystem-beta\nuser-beta\nshell-profile\nproject-setup\nshell-profile\nproject-setup' ]]
pass 'recipe entrypoints with explicit guest sudo precede overlays, environment, and project setup'

[[ $(stat -c %a "$HOME") == 750 && $(stat -c %a "$HOME/.bashrc") == 600 ]]
[[ $(stat -c %a "$HOME/.pi") == 700 && $(stat -c %a "$HOME/.config/vbx/environment.sh") == 600 ]]
[[ $(stat -c %u "$HOME/.bashrc") == "$(id -u)" ]]
[[ $(< "$HOME/unrelated") == existing && $(< "$HOME/.pi/agent/module-only") == keep ]]
[[ $(< "$HOME/.readonly/setting") == project ]]
[[ $(stat -c %a "$HOME/.readonly") == 500 && $(stat -c %a "$HOME/.readonly/setting") == 400 ]]
pass 'home merge preserves dotfiles, modes, home permissions, ownership, and unrelated files'

cp "$TEST_LOG" "$T/before"
printf 'exit 1\n' > "$T/bundle/project-setup.sh"
run_guest > /dev/null
cmp "$TEST_LOG" "$T/before"
pass 'complete marker prevents any repeat provisioning, overlays, or project setup'

# Host environment/config parsing, bundle transport, and argv tests live in
# internal/config, internal/provision, internal/workspace, and internal/process.

# Exercise the actual base script with every system write redirected to the fixture.
base=
started=false
while IFS= read -r line; do
    if [[ "$line" == '    script: |' ]]; then started=true
    elif [[ "$started" == true ]]; then base+="${line#      }"$'\n'; fi
done < "$ROOT/lima/default.yaml"
base=${base//\/var\/lib\/vbx/"$T/base-state"}
base=${base//\/var\/lib\/apt\/lists/"$T/apt-lists"}
[[ "$base" != *'/var/lib/'* ]]
printf '%s' "$base" > "$T/base.sh"
cat > "$T/bin/apt-get" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$TEST_ROOT/apt-log"
if [[ "$1" == upgrade && ${FAIL_UPGRADE:-0} == 1 ]]; then exit 1; fi
SH
chmod +x "$T/bin/apt-get"
bash "$T/base.sh"
bash "$T/base.sh"
[[ -f "$T/base-state/base-ready" && $(wc -l < "$T/apt-log") == 3 ]]
[[ $(< "$T/apt-log") == $'update\nupgrade -y --with-new-pkgs --no-remove\ninstall -y --no-install-recommends ca-certificates curl' ]]
pass 'base setup updates, upgrades, and installs in order, then skips repeated package operations'

rm "$T/base-state/base-ready" "$T/apt-log"
if FAIL_UPGRADE=1 bash "$T/base.sh" > /dev/null 2>&1; then exit 1; fi
[[ ! -f "$T/base-state/base-ready" && $(wc -l < "$T/apt-log") == 2 ]]
! grep -q '^install ' "$T/apt-log"
bash "$T/base.sh"
[[ -f "$T/base-state/base-ready" && $(wc -l < "$T/apt-log") == 5 ]]
pass 'failed upgrade blocks base installation and completion; retry resumes setup'

while IFS= read -r -d '' file; do bash -n "$file"; done < <(
    find "$ROOT/scripts" "$ROOT/modules" -name '*.sh' -print0
)
printf '%s provisioning offline checks passed.\n' "$passed"
