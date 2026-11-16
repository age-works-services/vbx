#!/usr/bin/env bash
# Exercise VBX provisioning retries with declarative persistence and synthetic modules.
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
T=$(mktemp -d)
trap 'rm -rf -- "$T"' EXIT
export HOME="$T/home" TEST_ROOT="$T"
mkdir -p "$HOME" "$T/bin" "$T/store" "$T/project" "$T/bundle/"{local-home/.tool,project-home,modules/installer,modules/sessions}
cp "$ROOT/scripts/persistence.sh" "$T/bundle/"
printf 'installer\nsessions\n' > "$T/bundle/modules.list"
printf 'sessions\t%s\tmount2\n' "$T/store" > "$T/bundle/bindings.tsv"
printf 'sessions\t~/.tool/sessions\tsessions\n' > "$T/bundle/mappings.tsv"
printf 'settings\n' > "$T/bundle/local-home/.tool/settings"
printf 'export ORDER=project\n' > "$T/bundle/environment.sh"
cat > "$T/bin/findmnt" <<'SH'
#!/bin/sh
printf '9p mount2 %s\n' "$3"
SH
cat > "$T/bin/sudo" <<'SH'
#!/bin/sh
[ "$1" = -n ] && [ "$2" = bash ] && [ "$3" = "$TEST_ROOT/bundle/guest-shell-system.sh" ]
SH
cat > "$T/bundle/modules/installer/setup.sh" <<'SH'
set -eu
[[ "$VBX_MODULE_DIR" == "$TEST_ROOT/bundle/modules/installer" ]]
[[ "$VBX_MODULE_DATA_DIR" == /mnt/vbx-data/installer ]]
printf 'installed\n' >> "$TEST_ROOT/install-log"
SH
cat > "$T/bundle/project-setup.sh" <<'SH'
set -eu
[[ "$ORDER" == project && -L "$HOME/.tool/sessions" ]]
[[ $(< "$HOME/.tool/settings") == settings ]]
printf 'session\n' > "$HOME/.tool/sessions/history"
[[ ! -e "$TEST_ROOT/fail" ]]
SH
chmod +x "$T/bin/"*
export PATH="$T/bin:$PATH"
touch "$T/fail"
if bash "$ROOT/scripts/provision-guest.sh" "$T/bundle" "$T/project" fixture > "$T/output" 2>&1; then exit 1; fi
state="$HOME/.local/state/vbx/provision/fixture"
[[ -f "$state/installer.done" && ! -e "$state/sessions.done" && ! -e "$state/complete" ]]
[[ $(< "$T/store/sessions/history") == session ]]
rm "$T/fail"
bash "$ROOT/scripts/provision-guest.sh" "$T/bundle" "$T/project" fixture > "$T/output" 2>&1
[[ -f "$state/complete" && $(wc -l < "$T/install-log") -eq 1 ]]
[[ $(readlink "$HOME/.tool/sessions") == "$T/store/sessions" ]]
printf 'PASS: persistence-only modules link before project setup and retain data across retries\n'
