#!/usr/bin/env bash
# Runs as root inside the guest.
set -euo pipefail
install -d -m 755 /etc/profile.d
cat > /etc/profile.d/vbx.sh <<'PROFILE'
case ":$PATH:" in
    *":$HOME/.local/bin:"*) ;;
    *) export PATH="$HOME/.local/bin:$PATH" ;;
esac
if [ -f "$HOME/.config/vbx/environment.sh" ]; then
    . "$HOME/.config/vbx/environment.sh"
fi
PROFILE
chmod 644 /etc/profile.d/vbx.sh
# Ubuntu reads this for interactive non-login Bash terminals, including VS Code.
if ! grep -Fq '# VBX environment' /etc/bash.bashrc; then
    printf '\n# VBX environment\n. /etc/profile.d/vbx.sh\n' >> /etc/bash.bashrc
fi
