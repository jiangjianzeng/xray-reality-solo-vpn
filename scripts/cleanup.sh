#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "${SCRIPT_DIR}/lib/i18n.sh"
select_lang_if_needed

timestamp="$(date +%Y%m%d-%H%M%S)"
backup_dir="/var/backups/xray-reality-solo-vpn-${timestamp}"

mkdir -p "$backup_dir"

systemctl stop xray-reality-solo-vpn xray caddy nginx 2>/dev/null || true
systemctl disable xray-reality-solo-vpn xray 2>/dev/null || true

for path in /opt/xray-reality-solo-vpn /etc/xray-reality-solo-vpn /var/lib/xray-reality-solo-vpn /root/vpn; do
  if [[ -e "$path" ]]; then
    cp -a "$path" "$backup_dir/" 2>/dev/null || true
    rm -rf "$path"
  fi
done

rm -f /etc/systemd/system/xray-reality-solo-vpn.service /etc/systemd/system/xray.service
systemctl daemon-reload

echo "$(translate cleanup_finished "$backup_dir")"
