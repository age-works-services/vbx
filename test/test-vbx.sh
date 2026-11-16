#!/usr/bin/env bash
# Offline CLI and lifecycle regression tests with an injected VM backend.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
go test ./internal/command ./internal/cli ./internal/commands ./internal/workspace ./internal/backend/lima
