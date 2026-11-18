#!/usr/bin/env bash
set -euo pipefail

fail() { printf 'Go module: %s\n' "$*" >&2; exit 1; }

# The project's native Go files own the toolchain selection. A root workspace
# takes precedence over the module, just as it does for the Go command.
requested=
for manifest in go.work go.mod; do
    [[ -f "$manifest" ]] || continue
    requested=$(awk '
        { sub(/\/\/.*/, "") }
        $1 == "go" { version = "go" $2 }
        $1 == "toolchain" && $2 != "default" { toolchain = $2 }
        END { print (toolchain != "" ? toolchain : version) }
    ' "$manifest")
    break
done
if [[ -n "$requested" && ! "$requested" =~ ^go[0-9]+\.[0-9]+(\.[0-9]+|(beta|rc)[0-9]+)?$ ]]; then
    fail "unsupported release '$requested' in $manifest (expected an official Go version)"
fi

case "$(uname -m)" in
    x86_64) arch=amd64 ;;
    aarch64) arch=arm64 ;;
    *) fail 'unsupported guest architecture' ;;
esac

root="$HOME/.local/share/vbx/go"
bin="$HOME/.local/bin"
mkdir -p -- "$root" "$bin"
work=$(mktemp -d "$root/.install.XXXXXXXX")
trap 'rm -rf -- "$work"' EXIT

curl -fsSL --retry 3 'https://go.dev/dl/?mode=json&include=all' -o "$work/releases.json"
# A language version without a patch (e.g. go 1.25) uses the latest patch
# release in that series. Without a manifest, use the latest stable release.
release=$(jq -er --arg requested "$requested" --arg arch "$arch" '
    [ .[] | .version as $version
      | select(
          if $requested == "" then .stable and ($version | test("^go[0-9]+\\.[0-9]+\\.[0-9]+$"))
          elif ($requested | test("^go[0-9]+\\.[0-9]+$")) then
              ($version == $requested or
               ($version | startswith($requested + ".") and test("^go[0-9]+\\.[0-9]+\\.[0-9]+$")))
          else $version == $requested
          end)
      | .files[] | select(.os == "linux" and .arch == $arch and .kind == "archive")
      | {version: $version, filename, sha256} ]
    | sort_by(.version | [scan("[0-9]+") | tonumber]) | last
    | select(. != null) | [.version, .filename, .sha256] | @tsv
' "$work/releases.json") || fail "no official Linux/$arch archive for ${requested:-latest stable Go}"
IFS=$'\t' read -r version filename checksum <<< "$release"
[[ "$version" =~ ^go[0-9]+\.[0-9]+(\.[0-9]+|(beta|rc)[0-9]+)?$ &&
   "$filename" == "$version.linux-$arch.tar.gz" && "$checksum" =~ ^[0-9a-f]{64}$ ]] ||
    fail 'invalid archive metadata'

destination="$root/$version"
valid_toolchain() {
    [[ -x "$1/bin/go" ]] &&
        [[ "$(GOTOOLCHAIN=local "$1/bin/go" version)" == "go version $version linux/$arch" ]]
}
if ! valid_toolchain "$destination"; then
    curl -fsSL --retry 3 "https://go.dev/dl/$filename" -o "$work/go.tar.gz"
    printf '%s  %s\n' "$checksum" "$work/go.tar.gz" | sha256sum --check --status || fail "checksum mismatch for $filename"
    tar -xzf "$work/go.tar.gz" -C "$work"
    valid_toolchain "$work/go" || fail "unexpected version in $filename"
    rm -rf -- "$destination"
    mv -T -- "$work/go" "$destination"
fi
ln -sfnT -- "$destination/bin/go" "$bin/go"
ln -sfnT -- "$destination/bin/gofmt" "$bin/gofmt"
"$bin/go" version
