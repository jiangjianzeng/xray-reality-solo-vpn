#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "${SCRIPT_DIR}/lib/i18n.sh"
select_lang_if_needed

echo
echo "== $(translate install_intro_title) =="
echo "$(translate install_intro_body)"
echo "$(translate install_intro_notice)"
echo

VERSION="${VERSION:-$(date +%Y%m%d-%H%M%S)}"
FORCE_CLEAN=0

if [[ "${1:-}" == "--clean" ]]; then
  FORCE_CLEAN=1
fi

prompt_required() {
  local key="$1"
  local prompt="$2"
  local default="${3:-}"
  local value=""
  while [[ -z "$value" ]]; do
    if [[ -n "$default" ]]; then
      read -r -p "${prompt} [${default}]: " value
      value="${value:-$default}"
    else
      read -r -p "${prompt}: " value
    fi
  done
  printf '%s' "$value"
}

detect_public_ip() {
  local value=""

  value="$(curl -4fsS --max-time 5 https://api.ipify.org 2>/dev/null || true)"
  if [[ -z "$value" ]]; then
    value="$(curl -4fsS --max-time 5 https://ifconfig.me/ip 2>/dev/null || true)"
  fi
  if [[ -z "$value" ]]; then
    value="$(curl -4fsS --max-time 5 https://icanhazip.com 2>/dev/null | tr -d '\r\n' || true)"
  fi

  printf '%s' "$value"
}

if ! "${ROOT_DIR}/scripts/check.sh"; then
  if [[ "$FORCE_CLEAN" -eq 1 ]]; then
    "${ROOT_DIR}/scripts/cleanup.sh"
  else
    exit 1
  fi
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y curl ca-certificates openssl nginx gnupg debian-keyring debian-archive-keyring apt-transport-https

if ! apt-cache show caddy >/dev/null 2>&1; then
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' > /etc/apt/sources.list.d/caddy-stable.list
  apt-get update
fi

apt-get install -y caddy

if ! command -v xray >/dev/null 2>&1; then
  bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install
fi

if [[ ! -x "${ROOT_DIR}/build/manager-linux-amd64" ]]; then
  echo "$(translate install_missing_manager)"
  echo "$(translate install_missing_artifact_hint)"
  exit 1
fi

if [[ ! -d "${ROOT_DIR}/web/dist" ]]; then
  echo "$(translate install_missing_web_dist)"
  echo "$(translate install_missing_artifact_hint)"
  exit 1
fi

PANEL_DOMAIN="$(prompt_required PANEL_DOMAIN "$(translate install_panel_domain)")"
LINE_DOMAIN="$(prompt_required LINE_DOMAIN "$(translate install_line_domain)")"
LINE_SERVER_ADDRESS_DEFAULT="$(detect_public_ip)"
if [[ -z "${LINE_SERVER_ADDRESS_DEFAULT}" ]]; then
  LINE_SERVER_ADDRESS_DEFAULT="$LINE_DOMAIN"
fi
LINE_SERVER_ADDRESS="$(prompt_required LINE_SERVER_ADDRESS "$(translate install_line_server_address)" "$LINE_SERVER_ADDRESS_DEFAULT")"
XRAY_REALITY_TARGET="$(prompt_required XRAY_REALITY_TARGET "$(translate install_reality_target)" 'www.cloudflare.com:443')"
ACME_EMAIL="$(prompt_required ACME_EMAIL "$(translate install_acme_email)")"
SETUP_TTL_MINUTES="$(prompt_required SETUP_TTL_MINUTES "$(translate install_setup_ttl)" '30')"

install -d /opt/xray-reality-solo-vpn/releases/"$VERSION"/bin
install -d /opt/xray-reality-solo-vpn/releases/"$VERSION"/web
install -d /etc/xray-reality-solo-vpn
install -d /var/lib/xray-reality-solo-vpn/generated
install -d /var/log/xray-reality-solo-vpn

install -m 0755 "${ROOT_DIR}/build/manager-linux-amd64" /opt/xray-reality-solo-vpn/releases/"$VERSION"/bin/manager
rm -rf /opt/xray-reality-solo-vpn/releases/"$VERSION"/web/dist
cp -R "${ROOT_DIR}/web/dist" /opt/xray-reality-solo-vpn/releases/"$VERSION"/web/
cp -R "${ROOT_DIR}/scripts" /opt/xray-reality-solo-vpn/releases/"$VERSION"/

KEY_OUTPUT="$(xray x25519 | tr -d '\r')"
XRAY_PRIVATE_KEY="$(printf '%s\n' "$KEY_OUTPUT" | awk -F': ' '/^Private key:/ {print $2}')"
XRAY_PUBLIC_KEY="$(printf '%s\n' "$KEY_OUTPUT" | awk -F': ' '/^Public key:/ {print $2}')"
if [[ -z "${XRAY_PRIVATE_KEY}" || -z "${XRAY_PUBLIC_KEY}" ]]; then
  XRAY_PRIVATE_KEY="$(printf '%s\n' "$KEY_OUTPUT" | awk -F': ' '/^PrivateKey:/ {print $2}')"
  XRAY_PUBLIC_KEY="$(printf '%s\n' "$KEY_OUTPUT" | awk -F': ' '/^Password:/ {print $2}')"
fi

cat >/etc/xray-reality-solo-vpn/app.env <<EOF
APP_HOST=127.0.0.1
APP_PORT=3000
TZ=UTC
PANEL_DOMAIN=${PANEL_DOMAIN}
PANEL_BASE_URL=https://${PANEL_DOMAIN}
LINE_DOMAIN=${LINE_DOMAIN}
LINE_SERVER_ADDRESS=${LINE_SERVER_ADDRESS}
LINE_PUBLIC_PORT=443
XRAY_LISTEN_PORT=2443
XRAY_API_LISTEN_PORT=10085
XRAY_REALITY_TARGET=${XRAY_REALITY_TARGET}
XRAY_REALITY_SERVER_NAMES=$(printf '%s' "${XRAY_REALITY_TARGET%%:*}")
XRAY_PRIVATE_KEY=${XRAY_PRIVATE_KEY}
XRAY_PUBLIC_KEY=${XRAY_PUBLIC_KEY}
SESSION_SECRET=$(openssl rand -hex 32)
TRUST_PROXY=true
DATA_DIR=/var/lib/xray-reality-solo-vpn
GENERATED_DIR=/var/lib/xray-reality-solo-vpn/generated
SETUP_TICKET_FILE=/var/lib/xray-reality-solo-vpn/setup-ticket.json
SETUP_TTL_MINUTES=${SETUP_TTL_MINUTES}
XRAY_SERVICE_NAME=xray
XRAY_EXECUTABLE=/usr/local/bin/xray
EOF
chmod 600 /etc/xray-reality-solo-vpn/app.env

ln -sfn /opt/xray-reality-solo-vpn/releases/"$VERSION" /opt/xray-reality-solo-vpn/current

cp "${ROOT_DIR}/deploy/systemd/xray-reality-solo-vpn.service" /etc/systemd/system/xray-reality-solo-vpn.service
cp "${ROOT_DIR}/deploy/systemd/xray.service" /etc/systemd/system/xray.service
rm -rf /etc/systemd/system/xray.service.d /etc/systemd/system/xray@.service.d

sed \
  -e "s#__PANEL_DOMAIN__#${PANEL_DOMAIN}#g" \
  -e "s#__ACME_EMAIL__#${ACME_EMAIL}#g" \
  "${ROOT_DIR}/deploy/caddy/Caddyfile.template" >/etc/caddy/Caddyfile

sed \
  -e "s#__PANEL_DOMAIN__#${PANEL_DOMAIN}#g" \
  -e "s#__XRAY_LISTEN_PORT__#2443#g" \
  "${ROOT_DIR}/deploy/nginx/nginx.conf.template" >/etc/nginx/nginx.conf

set -a
. /etc/xray-reality-solo-vpn/app.env
set +a

/opt/xray-reality-solo-vpn/current/bin/manager --setup-only

systemctl daemon-reload
systemctl enable xray-reality-solo-vpn xray caddy nginx
systemctl restart xray-reality-solo-vpn
systemctl restart xray
systemctl restart caddy
systemctl restart nginx

SETUP_URL="$(cd /opt/xray-reality-solo-vpn/current && set -a && . /etc/xray-reality-solo-vpn/app.env && set +a && /opt/xray-reality-solo-vpn/current/bin/manager --issue-setup-ticket | sed -n 's/^SETUP_URL=//p')"

echo
echo "$(translate install_panel_url): https://${PANEL_DOMAIN}"
echo "$(translate install_setup_url): ${SETUP_URL}"
echo "$(translate install_setup_expires): ${SETUP_TTL_MINUTES} $(translate install_minutes)"
echo "$(translate install_login_url): https://${PANEL_DOMAIN}/login"
echo "$(translate install_ticket_hint): sudo cat /var/lib/xray-reality-solo-vpn/setup-ticket.json"
echo "$(translate install_service_hint): systemctl status xray-reality-solo-vpn xray nginx caddy"
echo "$(translate install_finish_note)"
