#!/usr/bin/env bash
set -euo pipefail
guest_home=$1
# Ubuntu restricts user namespaces for unprofiled applications.
# Grant them only to Puppeteer's Chrome path; retain Chromium's own sandbox.
if [[ -r /sys/module/apparmor/parameters/enabled ]] &&
   grep -q '^Y' /sys/module/apparmor/parameters/enabled; then
    command -v apparmor_parser >/dev/null
    [[ "$guest_home" != *[\"\\]* ]] || { printf 'Unsupported guest home path\n' >&2; exit 1; }
    printf 'abi <abi/4.0>,\ninclude <tunables/global>\nprofile vbx-browser "%s/.cache/puppeteer/chrome/**/chrome" flags=(unconfined) {\n  userns,\n}\n' "$guest_home" > /etc/apparmor.d/vbx-browser
    apparmor_parser -r /etc/apparmor.d/vbx-browser
fi
