#!/usr/bin/env bash
# Offline tests of the module's installer orchestration. No host installation or network.
set -euo pipefail
MODULE=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
T=$(mktemp -d)
trap 'rm -rf -- "$T"' EXIT
mkdir -m 700 "$T/keyring"
gpg --batch --no-options --no-autostart --homedir "$T/keyring" \
    --import "$MODULE/aws-cli-signing-key.asc" > "$T/key-import.log" 2>&1
fingerprint=$(gpg --batch --no-options --no-autostart --homedir "$T/keyring" --with-colons \
    --show-keys "$MODULE/aws-cli-signing-key.asc" 2>/dev/null | awk -F: '$1=="fpr"{print $10; exit}')
[[ "$fingerprint" == FB5DB77FD5C118B80511ADA8A6310ACC4672475C ]]
printf 'PASS: bundled key matches the AWS CLI signing fingerprint\n'
mkdir "$T/bin" "$T/fixtures"
export MODULE FIXTURES="$T/fixtures"
cat > "$T/bin/curl" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
url= output=
while (($#)); do
    case $1 in
        -o) output=$2; shift 2 ;;
        --retry) shift 2 ;;
        -fsSL) shift ;;
        https://*) url=$1; shift ;;
        *) exit 91 ;;
    esac
done
case $url in
    https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip) step=archive ;;
    https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip.sig) step=signature ;;
    *) exit 92 ;;
esac
printf '%s\n' "$step" >> "$CASE_ROOT/events"
[[ ${FAIL:-} != "$step" ]] || exit 18
printf 'downloaded\n' > "$output"
SH
cat > "$T/bin/gpg" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
home=
while (($#)); do
    case $1 in
        --homedir) home=$2; shift 2 ;;
        --import|--verify) operation=$1; shift; break ;;
        *) shift ;;
    esac
done
[[ "$home" == "$TMPDIR/"*/gnupg ]]
if [[ $operation == --import ]]; then
    printf 'import\n' >> "$CASE_ROOT/events"
    [[ $1 == "$MODULE/aws-cli-signing-key.asc" ]]
    [[ ${FAIL:-} != import ]] || exit 19
    touch "$home/imported"
else
    printf 'verify\n' >> "$CASE_ROOT/events"
    [[ -f "$home/imported" && -s "$1" && -s "$2" ]]
    [[ ${FAIL:-} != verify ]] || exit 20
    touch "$home/verified"
fi
SH
cat > "$T/bin/unzip" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf 'extract\n' >> "$CASE_ROOT/events"
[[ $1 == -q && -s "$2" && $3 == -d && -f "$4/gnupg/verified" ]]
[[ ${FAIL:-} != extract ]] || exit 21
mkdir "$4/aws"
cp "$FIXTURES/installer" "$4/aws/install"
SH
cat > "$T/bin/sudo" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ $1 == -n ]]
shift
# Only execute the fixture extracted inside this test's temporary directory.
[[ $1 == "$TMPDIR/"*/aws/install ]]
exec "$@"
SH
cat > "$T/fixtures/installer" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf 'install\n' >> "$CASE_ROOT/events"
[[ $# == 5 && $1 == --install-dir && $2 == /usr/local/aws-cli && $3 == --bin-dir && $4 == /usr/local/bin && $5 == --update ]]
touch "$CASE_ROOT/partial-install"
[[ ${FAIL:-} != install ]] || exit 22
touch "$CASE_ROOT/installed"
SH
cat > "$T/bin/aws" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf 'version\n' >> "$CASE_ROOT/events"
[[ $1 == --version && -f "$CASE_ROOT/installed" ]]
[[ ${FAIL:-} != version ]] || exit 23
if [[ ${FAIL:-} == wrong-version ]]; then printf 'aws-cli/1.0\n'; else printf 'aws-cli/2.0 fixture\n'; fi
SH
chmod +x "$T/bin/"* "$T/fixtures/installer"
export PATH="$T/bin:$PATH"
run() {
    local expected=$1
    if bash "$MODULE/setup.sh" > "$CASE_ROOT/output" 2>&1; then
        [[ $expected == success ]] || { printf 'Unexpected success: %s\n' "$FAIL" >&2; exit 1; }
    else
        [[ $expected == failure ]] || { cat "$CASE_ROOT/output" >&2; exit 1; }
    fi
    # All download, extraction, and GnuPG workspaces must disappear on every exit.
    [[ -z $(find "$TMPDIR" -mindepth 1 -print -quit) ]]
}
for FAIL in success archive signature import verify extract install version wrong-version; do
    export FAIL CASE_ROOT="$T/$FAIL"
    mkdir -p "$CASE_ROOT/tmp"
    export TMPDIR="$CASE_ROOT/tmp"
    if [[ $FAIL == success ]]; then
        run success
        [[ $(< "$CASE_ROOT/events") == $'archive\nsignature\nimport\nverify\nextract\ninstall\nversion' ]]
        # Repeated setup uses AWS's update path and verifies the installed CLI again.
        run success
        [[ $(grep -c '^install$' "$CASE_ROOT/events") == 2 ]]
    else
        run failure
        case $FAIL in
            archive|signature|import|verify) ! grep -Eq '^(extract|install|version)$' "$CASE_ROOT/events" ;;
            extract) ! grep -Eq '^(install|version)$' "$CASE_ROOT/events" ;;
            install) ! grep -q '^version$' "$CASE_ROOT/events"
                FAIL=success run success
                [[ -f "$CASE_ROOT/installed" && $(grep -c '^install$' "$CASE_ROOT/events") == 2 ]]
                ;;
        esac
    fi
    printf 'PASS: AWS CLI installer scenario %s\n' "$(basename "$CASE_ROOT")"
done
