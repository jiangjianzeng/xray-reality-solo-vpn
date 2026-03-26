#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "${SCRIPT_DIR}/lib/i18n.sh"
select_lang_if_needed

BLOCKED=0

print_block() {
  printf 'BLOCKED: %s\n' "$1"
  BLOCKED=1
}

check_path() {
  local path="$1"
  if [[ -e "$path" ]]; then
    print_block "$(translate check_blocked_path "$path")"
  fi
}

check_service() {
  local service="$1"
  if systemctl list-unit-files 2>/dev/null | awk '{print $1}' | grep -qx "${service}.service"; then
    print_block "$(translate check_blocked_service "$service")"
  fi
}

check_port() {
  local port="$1"
  local result
  result="$(ss -lntp 2>/dev/null | awk -v p=":${port}" '$4 ~ p {print}' || true)"
  if [[ -n "$result" ]]; then
    print_block "$(translate check_blocked_port "$port" "$result")"
  fi
}

check_path /opt/xray-reality-solo-vpn
check_path /etc/xray-reality-solo-vpn
check_path /var/lib/xray-reality-solo-vpn
check_path /root/vpn
check_service xray-reality-solo-vpn
check_service xray
check_service caddy
check_service nginx

check_port 80
check_port 443
check_port 3000
check_port 8443
check_port 2443
check_port 10085

if [[ "$BLOCKED" -eq 1 ]]; then
  echo "$(translate check_fail_hint)"
  exit 1
fi

echo "$(translate check_pass)"
