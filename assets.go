package vbx

import "embed"

// Assets includes only public installation inputs.
//
//go:embed lima/default.yaml scripts/provision-guest.sh scripts/apt-install.sh scripts/guest-shell-system.sh scripts/npm-policy.sh scripts/persistence.sh all:modules
var Assets embed.FS
