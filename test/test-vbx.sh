#!/usr/bin/env bash
# Offline launcher integration tests. All Lima/editor commands use test doubles.
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
T=$(mktemp -d)
trap 'chmod -R u+rwX -- "$T"; rm -rf -- "$T"' EXIT
export VBX_TEST_ROOT="$T/app" VBX_TEST_FAKE="$T/fake" HOME="$T/home"
export XDG_CONFIG_HOME="$T/config" XDG_STATE_HOME="$T/state"
mkdir -p "$VBX_TEST_ROOT"/{scripts,lima} "$VBX_TEST_FAKE/bin" "$HOME" "$T/one/app" "$T/two/app" "$T/api"
cp "$ROOT/vbx" "$VBX_TEST_ROOT/vbx"
cp "$ROOT/scripts/"{host.sh,config.pl,apt-deps.pl,apt-install.sh,environment.pl,provision-host.sh,provision-guest.sh,guest-shell-system.sh} "$VBX_TEST_ROOT/scripts/"
cp -r "$ROOT/modules" "$VBX_TEST_ROOT/"
cp "$ROOT/scripts/npm-policy.sh" "$VBX_TEST_ROOT/scripts/"
cp "$ROOT/lima/default.yaml" "$VBX_TEST_ROOT/lima/"
export PATH="$VBX_TEST_FAKE/bin:$PATH"
cat > "$VBX_TEST_FAKE/bin/limactl" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf '%q ' "$@" >> "$VBX_TEST_FAKE/log"; printf '\n' >> "$VBX_TEST_FAKE/log"
case "$1" in
    list)
        case "$3" in
            '{{.Name}}') [[ ! -f "$VBX_TEST_FAKE/name" ]] || cat "$VBX_TEST_FAKE/name" ;;
            '{{.Status}}') printf '%s\n' "${VBX_TEST_STATUS:-Stopped}" ;;
            *) printf '%s\n' "$VBX_TEST_FAKE/ssh.config" ;;
        esac ;;
    validate) [[ ${VBX_TEST_FAIL_VALIDATE:-0} == 0 ]] ;;
    start)
        if [[ ${VBX_TEST_SPAWN_START:-0} == 1 ]]; then
            sleep 60 </dev/null >/dev/null 2>&1 &
            printf '%s\n' "$!" > "$VBX_TEST_FAKE/daemon-pid"
        fi
        [[ ${VBX_TEST_FAIL_START:-0} == 0 ]] || exit 1
        if [[ "$3" == --name=* ]]; then
            printf '%s\n' "${3#--name=}" > "$VBX_TEST_FAKE/name"
            cp "$4" "$VBX_TEST_FAKE/created.yaml"
        fi
        printf 'Host lima-test\n  HostName localhost\n' > "$VBX_TEST_FAKE/ssh.config"
        ;;
    shell)
        if [[ " $* " == *' vbx-provision '* ]]; then
            [[ ${VBX_TEST_FAIL_SETUP:-0} == 0 ]] || exit 1
            cat > "$VBX_TEST_FAKE/bundle.tar"
            printf 'setup\n' >> "$VBX_TEST_FAKE/setups"
        else printf '%s\0' "$@" > "$VBX_TEST_FAKE/guest-args"; fi ;;
    stop) [[ ${VBX_TEST_FAIL_STOP:-0} == 0 ]] ;;
    delete)
        [[ ${VBX_TEST_FAIL_DELETE:-0} == 0 ]] || exit 1
        rm -f "$VBX_TEST_FAKE/name" ;;
    *) exit 1 ;;
esac
SH
cat > "$VBX_TEST_FAKE/bin/code" <<'SH'
#!/usr/bin/env bash
printf '%s\0' "$@" > "$VBX_TEST_FAKE/code-args"
SH
cat > "$T/runner" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
source "$VBX_TEST_ROOT/vbx"
check_host() { :; }
# Only the test wrapper can bypass the real host check and approval prompt.
if [[ ${VBX_TEST_APPROVE:-0} == 1 ]]; then approve_project() { :; }; fi
vbx_main "$@"
SH
chmod +x "$VBX_TEST_FAKE/bin/"* "$T/runner"
run_vbx() { bash "$T/runner" --project "$T/one/app" "$@"; }
passed=0
pass() { passed=$((passed + 1)); printf 'PASS: %s\n' "$*"; }

bash -n "$ROOT/vbx"
perl -c "$ROOT/scripts/config.pl"
bash "$ROOT/vbx" --help > "$T/help"
grep -q -- '--workspace-only' "$T/help"
if run_vbx --vscode --terminal > /dev/null 2>&1; then exit 1; fi
if run_vbx --exec > /dev/null 2>&1; then exit 1; fi
if run_vbx --unknown > /dev/null 2>&1; then exit 1; fi
pass 'help and option validation'

first=$(run_vbx --print-name)
second=$(bash "$T/runner" --project "$T/two/app" --print-name)
[[ "$first" != "$second" && "$first" == vbx-app-* ]]
ln -s "$T/one/app" "$T/link"
[[ $(bash "$T/runner" --project "$T/link" --print-name) == "$first" ]]
pass 'canonical identity distinguishes equal basenames'

run_vbx --render > "$T/default.yaml"
grep -q 'mountPoint: '\''/workspaces/app'\''' "$T/default.yaml"
grep -q 'cpus: 2' "$T/default.yaml"
grep -Fxq '  name: dev' "$T/default.yaml"
grep -Fxq '  home: /home/dev' "$T/default.yaml"
! grep -Eq '^  uid:' "$T/default.yaml"
grep -q 'resolute/release/ubuntu-26.04-minimal-cloudimg-amd64.img' "$T/default.yaml"
grep -Fq 'apt-get upgrade -y --with-new-pkgs --no-remove' "$T/default.yaml"
[[ ! -e "$VBX_TEST_FAKE/log" ]]
pass 'offline default rendering needs no Lima'

mkdir -p "$VBX_TEST_ROOT/config.local" "$T/one/app/.vbx"
cat > "$VBX_TEST_ROOT/config.local/config.yaml" <<'YAML'
version: 1
resources:
  cpus: 3
  memory: 6GiB
launch:
  default: workspace-only
YAML
cat > "$T/one/app/.vbx/config.yaml" <<'YAML'
version: 1
resources:
  cpus: 4
mounts:
  - source: ../../api
    target: /workspaces/api
    readonly: true
YAML
run_vbx --render > "$T/merged.yaml"
grep -q 'cpus: 4' "$T/merged.yaml"
grep -q 'memory: 6GiB' "$T/merged.yaml"
grep -q "location: '$T/api'" "$T/merged.yaml"
grep -q 'writable: false' "$T/merged.yaml"
pass 'scalar precedence, project-relative paths, read-only additional mounts'

cp "$T/one/app/.vbx/config.yaml" "$T/good.yaml"
printf 'version: 1\nunknown: true\n' > "$T/one/app/.vbx/config.yaml"
if run_vbx --render > /dev/null 2>&1; then exit 1; fi
printf 'version: 1\nresources:\n  cpus: 0\n' > "$T/one/app/.vbx/config.yaml"
if run_vbx --render > /dev/null 2>&1; then exit 1; fi
printf 'version: 1\nversion: 1\n' > "$T/one/app/.vbx/config.yaml"
if run_vbx --render > /dev/null 2>&1; then exit 1; fi
printf 'version: 1\nresources:\n\tcpus: 4\n' > "$T/one/app/.vbx/config.yaml"
if run_vbx --render > /dev/null 2>&1; then exit 1; fi
cp "$T/good.yaml" "$T/one/app/.vbx/config.yaml"
pass 'unknown keys, bad resources, duplicate keys, tabs fail'

sed 's@target: /workspaces/api@target: /workspaces/app@' "$T/good.yaml" > "$T/one/app/.vbx/config.yaml"
if run_vbx --render > /dev/null 2>&1; then exit 1; fi
sed 's@source: ../../api@source: .@' "$T/good.yaml" > "$T/one/app/.vbx/config.yaml"
if run_vbx --render > /dev/null 2>&1; then exit 1; fi
cp "$T/good.yaml" "$T/one/app/.vbx/config.yaml"
pass 'duplicate targets and overlapping host mounts fail'

if run_vbx --workspace-only < /dev/null > /dev/null 2>&1; then exit 1; fi
! grep -q '^start ' "$VBX_TEST_FAKE/log"
pass 'noninteractive project approval blocks creation'

: > "$VBX_TEST_FAKE/log"
if VBX_TEST_APPROVE=1 VBX_TEST_FAIL_VALIDATE=1 run_vbx --workspace-only > /dev/null 2>&1; then exit 1; fi
! grep -q '^start ' "$VBX_TEST_FAKE/log"
pass 'failed Lima validation blocks creation'

if VBX_TEST_APPROVE=1 VBX_TEST_FAIL_START=1 run_vbx --workspace-only > /dev/null 2>&1; then exit 1; fi
[[ $(cat "$XDG_STATE_HOME/vbx/instances/$first/status") == creating ]]
pass 'failed startup retains a non-ready snapshot'

VBX_TEST_APPROVE=1 run_vbx --workspace-only > "$T/output"
grep -q 'Ready:' "$T/output"
grep -q -- "--name=$first" "$VBX_TEST_FAKE/log"
[[ $(cat "$XDG_STATE_HOME/vbx/instances/$first/status") == ready ]]
[[ -L "$XDG_CONFIG_HOME/vbx/ssh/$first.conf" ]]
[[ $(wc -l < "$VBX_TEST_FAKE/setups") == 1 ]]
[[ ! -f "$XDG_STATE_HOME/vbx/instances/$first/provision.tar" ]]
pass 'approved create provisions once, persists state, and registers SSH without an agent'

# Reuse must ignore even invalid edits to project configuration.
printf 'broken: yaml\n' > "$T/one/app/.vbx/config.yaml"
: > "$VBX_TEST_FAKE/log"
run_vbx --workspace-only > /dev/null
! grep -q '^validate ' "$VBX_TEST_FAKE/log"
! grep -q -- '--name=' "$VBX_TEST_FAKE/log"
[[ $(wc -l < "$VBX_TEST_FAKE/setups") == 1 ]]
pass 'reuse retains saved configuration and skips setup and project approval'

run_vbx > "$T/default-output"
grep -q 'Ready:' "$T/default-output"
pass 'default launch action comes from the saved configuration'

(
    exec 8> "$XDG_STATE_HOME/vbx/locks/$first.lock"
    flock -n 8
    if run_vbx --workspace-only > /dev/null 2>&1; then exit 1; fi
)
pass 'concurrent lifecycle operation is rejected'

run_vbx --exec printf '%s' 'a b; $(touch /unexpected)' > /dev/null
mapfile -d '' -t args < "$VBX_TEST_FAKE/guest-args"
[[ ${args[1]} == --workdir=/workspaces/app && ${args[9]} == printf ]]
[[ ${args[11]} == 'a b; $(touch /unexpected)' ]]
pass 'explicit command forwards literal arguments'

run_vbx --terminal > /dev/null
mapfile -d '' -t args < "$VBX_TEST_FAKE/guest-args"
[[ ${args[9]} == bash && ${args[10]} == -l ]]
pass 'terminal launches a login shell in the primary workspace'

run_vbx --vscode > /dev/null
mapfile -d '' -t args < "$VBX_TEST_FAKE/code-args"
[[ ${args[0]} == --remote && ${args[1]} == "ssh-remote+lima-$first" && ${args[2]} == /workspaces ]]
run_vbx --ssh-config > "$T/include"
grep -Fq "$XDG_CONFIG_HOME/vbx/ssh/*.conf" "$T/include"
pass 'VS Code uses the SSH alias and opens the common workspace parent'

run_vbx --stop > /dev/null
if run_vbx --remove < /dev/null > /dev/null 2>&1; then exit 1; fi
[[ -f "$VBX_TEST_FAKE/name" ]]
pass 'stop works; removal refuses noninteractive confirmation'

# Literal shell syntax in configuration must never execute on the host.
cp "$T/good.yaml" "$T/one/app/.vbx/config.yaml"
printf '\nlaunch:\n  folder: /workspaces/$(touch %s)\n' "$T/pwned" >> "$T/one/app/.vbx/config.yaml"
run_vbx --render > "$T/literal.yaml"
[[ ! -e "$T/pwned" ]]
pass 'configuration is parsed as data, never sourced or evaluated'

# Quoted source paths must round-trip through the restricted parser.
special="$T/a 'quoted' directory"
mkdir -p "$special"
printf 'version: 1\nmounts:\n  - source: "../../a '\''quoted'\'' directory"\n    target: /workspaces/quoted\n' > "$T/one/app/.vbx/config.yaml"
run_vbx --render > "$T/quoted.yaml"
grep -Fq "location: '$T/a ''quoted'' directory'" "$T/quoted.yaml"
pass 'quoted mount paths preserve spaces and apostrophes'

# Test post-confirmation removal without requiring a real terminal or VM.
cat > "$T/remove-runner" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
source "$VBX_TEST_ROOT/vbx"
NAME=$1
STATE="$XDG_STATE_HOME/vbx/instances/$NAME"
SSH_DIR="$XDG_CONFIG_HOME/vbx/ssh"
remove_vm
SH
prepare_removal() {
    mkdir -p "$XDG_STATE_HOME/vbx/instances/$first" "$XDG_CONFIG_HOME/vbx/ssh"
    printf 'marker\n' > "$XDG_STATE_HOME/vbx/instances/$first/config"
    ln -sfn "$VBX_TEST_FAKE/ssh.config" "$XDG_CONFIG_HOME/vbx/ssh/$first.conf"
    printf '%s\n' "$first" > "$VBX_TEST_FAKE/name"
    : > "$VBX_TEST_FAKE/log"
}
prepare_removal
VBX_TEST_STATUS=Stopped bash "$T/remove-runner" "$first"
! grep -q '^stop ' "$VBX_TEST_FAKE/log"
grep -q '^delete ' "$VBX_TEST_FAKE/log"
[[ ! -e "$XDG_STATE_HOME/vbx/instances/$first" && ! -L "$XDG_CONFIG_HOME/vbx/ssh/$first.conf" ]]
[[ -d "$T/one/app" ]]
pass 'stopped VM removal skips stop and cleans only VBX state'

prepare_removal
VBX_TEST_STATUS=Running bash "$T/remove-runner" "$first"
[[ $(grep -E '^(stop|delete) ' "$VBX_TEST_FAKE/log" | cut -d ' ' -f 1) == $'stop\ndelete' ]]
pass 'running VM removal stops before deletion'

prepare_removal
if VBX_TEST_STATUS=Running VBX_TEST_FAIL_STOP=1 bash "$T/remove-runner" "$first" > /dev/null 2>&1; then exit 1; fi
! grep -q '^delete ' "$VBX_TEST_FAKE/log"
[[ -f "$XDG_STATE_HOME/vbx/instances/$first/config" && -L "$XDG_CONFIG_HOME/vbx/ssh/$first.conf" ]]
pass 'failed stop blocks deletion and preserves VBX state'

prepare_removal
if VBX_TEST_FAIL_DELETE=1 bash "$T/remove-runner" "$first" > /dev/null 2>&1; then exit 1; fi
[[ -f "$XDG_STATE_HOME/vbx/instances/$first/config" && -L "$XDG_CONFIG_HOME/vbx/ssh/$first.conf" ]]
pass 'failed deletion preserves VBX state'

prepare_removal
if VBX_TEST_STATUS=Broken bash "$T/remove-runner" "$first" > /dev/null 2>&1; then exit 1; fi
! grep -Eq '^(stop|delete) ' "$VBX_TEST_FAKE/log"
pass 'unexpected VM status blocks destructive operations'

# Creation-only lifecycle regression: a failed setup retains only the approved inputs.
rm -f "$VBX_TEST_FAKE/name"
printf 'version: 1\nmodules:\n  - pi\napt:\n  packages:\n    - "bash (>= 5)"\nenvFiles:\n  - .env\n' > "$T/one/app/.vbx/config.yaml"
printf 'TEST_TOKEN=initial-secret\n' > "$T/one/app/.env"
mkdir -p "$VBX_TEST_ROOT/config.local/home/.readonly" "$T/one/app/.vbx/home"
printf 'local-home\n' > "$VBX_TEST_ROOT/config.local/home/.readonly/setting"
chmod 500 "$VBX_TEST_ROOT/config.local/home/.readonly"
printf 'original-home\n' > "$T/one/app/.vbx/home/.setting"
printf 'echo original-setup\ntouch %q\n' "$T/host-executed" > "$T/one/app/.vbx/setup.sh"
rm -f "$VBX_TEST_FAKE/code-args"
if VBX_TEST_APPROVE=1 VBX_TEST_FAIL_SETUP=1 VBX_TEST_SPAWN_START=1 run_vbx --vscode > "$T/failure-log" 2>&1; then exit 1; fi
state="$XDG_STATE_HOME/vbx/instances/$first"
[[ $(< "$state/status") == setup-failed && -f "$state/provision.tar" ]]
[[ $(stat -c %a "$state/provision.tar") == 600 && $(stat -c %a "$state") == 700 ]]
! grep -q 'initial-secret' "$T/failure-log"
grep -q 'incomplete VM is kept for inspection' "$T/failure-log"
grep -Fq -- '--remove' "$T/failure-log"
grep -Fq -- '--workspace-only' "$T/failure-log"
! grep -q 'original saved inputs' "$T/failure-log"
! grep -qi 'back.*up\|save.*guest' "$T/failure-log"
[[ ! -e "$VBX_TEST_FAKE/code-args" && ! -e "$T/host-executed" ]]
cp "$state/provision.tar" "$T/approved.tar"
pass 'failed setup blocks launch, preserves private snapshot, and does not log secrets'

daemon=$(< "$VBX_TEST_FAKE/daemon-pid")
kill -0 "$daemon"
if [[ -e "/proc/$daemon/fd/9" ]] || ! flock -n "$XDG_STATE_HOME/vbx/locks/$first.lock" true; then
    kill "$daemon"
    exit 1
fi
kill "$daemon"
pass 'failed setup releases its lock and persistent Lima children never inherit the lock descriptor'

printf 'changed: invalid\n' > "$T/one/app/.vbx/config.yaml"
printf 'TEST_TOKEN=changed-secret\n' > "$T/one/app/.env"
printf 'echo changed-setup\n' > "$T/one/app/.vbx/setup.sh"
printf 'changed-home\n' > "$T/one/app/.vbx/home/.setting"
run_vbx --workspace-only > /dev/null
cmp "$T/approved.tar" "$VBX_TEST_FAKE/bundle.tar"
[[ $(< "$state/status") == ready && ! -f "$state/provision.tar" ]]
mkdir "$T/archive"
tar -xf "$T/approved.tar" -C "$T/archive"
grep -q 'original-setup' "$T/archive/project-setup.sh"
grep -q 'initial-secret' "$T/archive/environment.sh"
[[ $(< "$T/archive/project-home/.setting") == original-home ]]
[[ $(stat -c %a "$VBX_TEST_ROOT/config.local/home/.readonly") == 500 ]]
[[ $(< "$T/archive/modules.list") == pi ]]
grep -Fxq git "$T/archive/apt.txt"
grep -Fxq 'bash (>= 5)' "$T/archive/apt.txt"
! grep -Eq '^(build-essential|postgresql-client|vim-tiny|python3|python3-venv|pipx|apparmor|libgtk-3-0t64)$' "$T/archive/apt.txt"
pass 'retry uses frozen setup, environment and dependency order; success removes pending archive'

# Invalid recipe package declarations are rejected before creating a guest.
rm -f "$VBX_TEST_FAKE/name"
printf 'version: 1\nmodules:\n  - pi\n' > "$T/one/app/.vbx/config.yaml"
printf '\n--allow-unauthenticated\n' >> "$VBX_TEST_ROOT/modules/pi/apt.txt"
: > "$VBX_TEST_FAKE/log"
if VBX_TEST_APPROVE=1 run_vbx --workspace-only > /dev/null 2>&1; then exit 1; fi
! grep -q '^start ' "$VBX_TEST_FAKE/log"
pass 'invalid module APT requirements block creation before guest startup'

printf '%s launcher offline checks passed.\n' "$passed"
