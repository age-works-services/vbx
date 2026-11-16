#!/usr/bin/env bash
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
install -d -m 755 /etc/apt/keyrings
key=$(mktemp)
trap 'rm -f "$key"' EXIT
curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key -o "$key"
gpg --batch --yes --dearmor -o /etc/apt/keyrings/nodesource.gpg "$key"
chmod 644 /etc/apt/keyrings/nodesource.gpg
printf '%s\n' 'deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_24.x nodistro main' > /etc/apt/sources.list.d/nodesource.list
apt-get update
apt-get install -y --no-remove --no-install-recommends nodejs
rm -rf /var/lib/apt/lists/*
