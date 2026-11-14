#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
T=$(mktemp -d)
trap 'rm -rf -- "$T"' EXIT
# No module may reach into the framework or a sibling module for installer code.
if grep -REn '\$support|/support/|\.\./|scripts/recipes|VBX_ROOT' "$ROOT/modules" --include='*.sh' --include='*.cjs'; then exit 1; fi
printf 'PASS: module installers contain no framework/sibling installer references\n'

export HOME="$T/home" TEST_MODULE="$T/isolated node" TEST_LOG="$T/log"
mkdir -p "$HOME" "$T/bin"
cp -a "$ROOT/modules/node" "$TEST_MODULE"
cat > "$T/bin/node" <<'SH'
#!/bin/bash
# Simulate a toolchain that needs installation.
exit 1
SH
cat > "$T/bin/npm" <<'SH'
#!/bin/bash
case "$1" in
    --version) printf '11.19.1\n';;
    config) printf 'npm %s\n' "$*" >> "$TEST_LOG";;
    *) exit 99;;
esac
SH
cat > "$T/bin/sudo" <<'SH'
#!/bin/bash
set -eu
[[ "$1" == -n && "$2" == bash && "$3" == "$TEST_MODULE/system.sh" ]]
[[ -f "$3" ]]
printf 'own-system-installer\n' >> "$TEST_LOG"
# Never run privileged package/file operations in this offline fixture.
SH
chmod +x "$T/bin/"*
PATH="$T/bin:/usr/bin:/bin" bash "$TEST_MODULE/setup.sh"
grep -Fxq own-system-installer "$TEST_LOG"
grep -Fq "npm config set prefix $HOME/.local --location=user" "$TEST_LOG"
[[ ! -e "$T/support" && ! -e "$T/scripts" ]]
printf 'PASS: isolated node module invokes its own installer and npm policy without external recipe files\n'

# Pi must fail on a missing prerequisite instead of silently installing Node.
cp -a "$ROOT/modules/pi" "$T/pi"
rm "$T/bin/npm"
ln -s "$(command -v dirname)" "$T/bin/dirname"
cp "$TEST_LOG" "$T/before"
if PATH="$T/bin" /bin/bash "$T/pi/setup.sh" > "$T/pi-error" 2>&1; then exit 1; fi
grep -q 'select node before pi' "$T/pi-error"
cmp "$TEST_LOG" "$T/before"
printf 'PASS: Pi reports missing Node/npm without invoking another installer\n'
printf '3 module boundary checks passed.\n'
