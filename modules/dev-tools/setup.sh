#!/usr/bin/env bash
set -euo pipefail
# Packages are already installed centrally. Only optional completion setup remains.
sudo -n bash -c 'gh completion -s bash > /etc/bash_completion.d/gh'
