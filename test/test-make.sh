#!/usr/bin/env bash
# Exercise Make installation in an isolated checkout with a fake Go compiler.
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
T=$(mktemp -d)
trap 'rm -rf -- "$T"' EXIT
mkdir -p "$T/src/test" "$T/tools"
cp "$ROOT/Makefile" "$T/src/Makefile"
export TEST_MAKE_LOG="$T/builds"
cat > "$T/tools/go" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
    'build -trimpath -o bin/vbx ./cmd/vbx')
        [[ "$CGO_ENABLED" == 0 ]]
        printf 'build\n' >> "$TEST_MAKE_LOG"
        [[ ${TEST_FAIL_BUILD:-0} == 0 ]]
        mkdir -p bin
        printf '#!/bin/sh\nprintf "installed vbx\\n"\n' > bin/vbx
        ;;
    'test ./...')
        [[ -f bin/vbx && ! -e "$TEST_INSTALL_PATH" ]]
        printf 'test\n' >> "$TEST_MAKE_LOG"
        [[ ${TEST_FAIL_TEST:-0} == 0 ]]
        ;;
    *) exit 99 ;;
esac
SH
cat > "$T/src/test/test-fixture.sh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ -f bin/vbx && ! -e "$TEST_INSTALL_PATH" ]]
printf 'shell-test\n' >> "$TEST_MAKE_LOG"
SH
chmod +x "$T/tools/go"
export PATH="$T/tools:$PATH"
# Keep nested make independent of the outer invocation's goals and variables.
unset MAKEFLAGS MFLAGS MAKEOVERRIDES PREFIX BINDIR DESTDIR
cd "$T/src"
help=$(make --no-print-directory help)
for target in all help build install test check cross-check PREFIX BINDIR DESTDIR; do
    [[ "$help" == *"$target"* ]]
done
[[ ! -e "$TEST_MAKE_LOG" && ! -e bin/vbx ]]
printf 'PASS: make help lists targets and options without building\n'
for mode in serial parallel; do
    rm -f "$TEST_MAKE_LOG" bin/vbx
    export TEST_INSTALL_PATH="$T/$mode/bin/vbx"
    args=(); [[ $mode != parallel ]] || args=(-j4)
    make --no-print-directory "${args[@]}" BINDIR="$T/$mode/bin"
    [[ -x "$TEST_INSTALL_PATH" ]]
    cmp bin/vbx "$TEST_INSTALL_PATH"
    # install retains its build prerequisite when invoked directly or recursively.
    [[ $(< "$TEST_MAKE_LOG") == $'build\ntest\nshell-test\nbuild' ]]
done
printf 'PASS: bare make builds, tests, then installs, including with -j\n'
for failure in BUILD TEST; do
    rm -f "$TEST_MAKE_LOG" bin/vbx
    export TEST_INSTALL_PATH="$T/fail-$failure/bin/vbx"
    if env "TEST_FAIL_$failure=1" make --no-print-directory -j4 BINDIR="$T/fail-$failure/bin"; then
        printf 'FAIL: make ignored a %s failure\n' "$failure" >&2
        exit 1
    fi
    [[ ! -e "$TEST_INSTALL_PATH" ]]
    expected=build
    [[ $failure != TEST ]] || expected=$'build\ntest'
    [[ $(< "$TEST_MAKE_LOG") == "$expected" ]]
done
printf 'PASS: build and test failures prevent installation\n'
make --no-print-directory install HOME="$T/home with spaces"
default="$T/home with spaces/.local/bin/vbx"
[[ -x "$default" && $(stat -c '%a' "$default") == 755 ]]
cmp bin/vbx "$default"
[[ $("$default") == 'installed vbx' ]]
printf 'PASS: install builds and installs an executable under HOME\n'
make --no-print-directory install PREFIX="$T/custom prefix"
cmp bin/vbx "$T/custom prefix/bin/vbx"
make --no-print-directory install PREFIX=/unused BINDIR="$T/custom bin"
cmp bin/vbx "$T/custom bin/vbx"
make --no-print-directory install PREFIX=/usr/local DESTDIR="$T/staging root"
cmp bin/vbx "$T/staging root/usr/local/bin/vbx"
printf 'PASS: PREFIX, BINDIR, and DESTDIR overrides support paths with spaces\n'
