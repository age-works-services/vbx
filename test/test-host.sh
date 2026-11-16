#!/usr/bin/env bash
# Offline host logic tests. Do not inspect private configuration or start VMs.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
go test ./internal/project ./internal/state ./internal/process ./internal/host ./internal/interaction
