#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "${SCRIPT_DIR}/lib/i18n.sh"
select_lang_if_needed

cd /opt/xray-reality-solo-vpn/current
set -a
. /etc/xray-reality-solo-vpn/app.env
set +a

/opt/xray-reality-solo-vpn/current/bin/manager --issue-setup-ticket
echo "$(translate reset_setup_usage)"
