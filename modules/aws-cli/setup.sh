#!/usr/bin/env bash
set -euo pipefail
here=$(cd -- "$(dirname -- "$0")" && pwd -P)
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
mkdir -m 700 "$work/gnupg"
url=https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip
curl -fsSL --retry 3 "$url" -o "$work/awscliv2.zip"
curl -fsSL --retry 3 "$url.sig" -o "$work/awscliv2.zip.sig"
# Trust only the bundled AWS signing key, without touching the user's keyring.
gpg=(gpg --batch --no-options --no-autostart --homedir "$work/gnupg")
"${gpg[@]}" --import "$here/aws-cli-signing-key.asc"
"${gpg[@]}" --no-auto-key-retrieve --verify "$work/awscliv2.zip.sig" "$work/awscliv2.zip"
unzip -q "$work/awscliv2.zip" -d "$work"
# --update also supports a fresh installation and makes incomplete setup retryable.
sudo -n "$work/aws/install" --install-dir /usr/local/aws-cli --bin-dir /usr/local/bin --update
version=$(aws --version)
[[ "$version" == aws-cli/2.* ]] || { printf 'AWS CLI v2 required; found %s\n' "$version" >&2; exit 1; }
printf '%s\n' "$version"
