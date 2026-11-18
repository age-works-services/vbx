#!/usr/bin/env bash
set -euo pipefail
setup=$(cd -- "$(dirname -- "$0")/.." && pwd)/setup.sh
fixture=$(mktemp -d)
trap 'rm -rf -- "$fixture"' EXIT
mkdir -p "$fixture/bin" "$fixture/project" "$fixture/archive/go/bin"
export fixture
export PATH="$fixture/bin:$PATH"
export GOTOOLCHAIN=local

cat > "$fixture/bin/uname" <<'SH'
#!/bin/bash
[[ $1 == -m ]]
echo x86_64
SH
cat > "$fixture/bin/curl" <<'SH'
#!/bin/bash
set -eu
[[ $1 == -fsSL && $2 == --retry && $3 == 3 && $5 == -o ]]
case "$4" in
    'https://go.dev/dl/?mode=json&include=all') cp "$fixture/releases.json" "$6" ;;
    https://go.dev/dl/go*.linux-amd64.tar.gz) cp "$fixture/${4##*/}" "$6" ;;
    *) exit 1 ;;
esac
SH
chmod +x "$fixture/bin/"*

make_release() {
    local version=$1 stable=$2 filename="$1.linux-amd64.tar.gz" hash
    printf '#!/bin/bash\n[[ $1 == version ]] || exit 1\necho "go version %s linux/amd64"\n' "$version" > "$fixture/archive/go/bin/go"
    cp "$fixture/archive/go/bin/go" "$fixture/archive/go/bin/gofmt"
    chmod +x "$fixture/archive/go/bin/"*
    tar -czf "$fixture/$filename" -C "$fixture/archive" go
    hash=$(sha256sum "$fixture/$filename")
    jq -n --arg version "$version" --argjson stable "$stable" --arg filename "$filename" --arg sha256 "${hash%% *}" \
        '{ $version, $stable, files: [{os: "linux", arch: "amd64", kind: "archive", $filename, $sha256}] }'
}
{
    make_release go1.25.10 false
    make_release go1.26.0 true
    make_release go1.25.2 false
    make_release go1.27rc1 false
} | jq -s . > "$fixture/releases.json"

cd "$fixture/project"
check_version() {
    export HOME="$fixture/home-$1"
    [[ "$(bash "$setup")" == "go version $2 linux/amd64" ]]
    [[ -x "$HOME/.local/bin/gofmt" ]]
}
check_version default go1.26.0
printf 'module example.org/project\n\ngo 1.25 // trailing comment\n' > go.mod
check_version minor go1.25.10
printf '\ntoolchain go1.25.2\n' >> go.mod
check_version toolchain go1.25.2
printf 'go 1.27rc1\n\nuse .\n' > go.work
check_version workspace go1.27rc1

export HOME="$fixture/home-corrupt"
printf 'invalid archive\n' > "$fixture/go1.27rc1.linux-amd64.tar.gz"
if bash "$setup" > "$fixture/output" 2>&1; then
    printf 'Accepted a corrupt Go archive\n' >&2
    exit 1
fi
grep -q 'checksum mismatch' "$fixture/output"
[[ ! -e "$HOME/.local/bin/go" && ! -L "$HOME/.local/bin/go" ]]
