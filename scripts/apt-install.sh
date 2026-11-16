#!/usr/bin/env bash
# Runs as root inside the guest. APT owns dependency and version resolution.
set -euo pipefail
mapfile -t requirements < "$1"
[[ ${#requirements[@]} -gt 0 ]] || exit 0
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get satisfy -y --no-remove --no-install-recommends -- "${requirements[@]}"
rm -rf /var/lib/apt/lists/*
