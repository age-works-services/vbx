#!/usr/bin/env bash
# Offline tests for VBX's APT argument forwarding and provisioning failure handling.
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
T=$(mktemp -d)
trap 'chmod -R u+rwX -- "$T"; rm -rf -- "$T"' EXIT
mkdir -p "$T/project" "$T/bin" "$T/home"
export TEST_ROOT="$T" HOME="$T/home"
passed=0
pass() { passed=$((passed + 1)); printf 'PASS: %s\n' "$*"; }

# Host APT/config validation lives in internal/config and internal/provision.
printf 'git\nexample-tool (>= 1.0)\nexample-tool (<= 2.0)\n' > "$T/apt.txt"

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
[[ ${args[6]} == 'example-tool (>= 1.0)' && ${args[7]} == 'example-tool (<= 2.0)' ]]
if FAIL_APT=1 PATH="$T/bin:$PATH" bash "$T/apt-install.sh" "$T/apt.txt" > /dev/null 2>&1; then exit 1; fi
pass 'APT receives literal native constraints with no removals, and solver failures propagate'

# Guest APT failure must block every module and retain an incomplete state.
mkdir -p "$T/bundle/"{local-home,project-home,modules/check}
cp "$T/apt-install.sh" "$T/bundle/apt-install.sh"
cp "$T/apt.txt" "$T/bundle/apt.txt"
cp "$ROOT/scripts/guest-shell-system.sh" "$T/bundle/"
printf 'check\n' > "$T/bundle/modules.list"
printf 'export SETUP_INPUT=ready\n' > "$T/bundle/environment.sh"
cat > "$T/bundle/project-setup.sh" <<'SH'
set -eu
[[ "$SETUP_INPUT" == ready ]]
SH
printf 'touch "$TEST_ROOT/module-ran"\n' > "$T/bundle/modules/check/setup.sh"
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
[[ ! -e "$T/module-ran" && ! -f "$state/apt.done" && ! -f "$state/complete" ]]
guest > "$T/guest-log"
[[ -f "$T/module-ran" && -f "$state/apt.done" && -f "$state/complete" ]]
cp "$T/apt-log" "$T/previous-apt"
guest > /dev/null
cmp "$T/apt-log" "$T/previous-apt"
pass 'failed APT blocks module/launch readiness; retry completes and ordinary reuse skips setup'

printf '%s APT provisioning checks passed.\n' "$passed"
