#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "${SCRIPT_DIR}/lib/i18n.sh"
select_lang_if_needed

echo
echo "== $(translate update_intro_title) =="
echo "$(translate update_intro_body)"
echo "$(translate update_intro_notice)"
echo

if [[ "${SKIP_ROOT_CHECK:-0}" != "1" && "${EUID:-$(id -u)}" -ne 0 ]]; then
  echo "$(translate update_root_required)"
  exit 1
fi

INSTALL_ROOT="${INSTALL_ROOT:-/opt/xray-reality-solo-vpn}"
SERVICE_NAME="${SERVICE_NAME:-xray-reality-solo-vpn}"
SYSTEMCTL_BIN="${SYSTEMCTL_BIN:-systemctl}"
VERSION="${VERSION:-$(date +%Y%m%d-%H%M%S)}"
RELEASE_DIR="${INSTALL_ROOT}/releases/${VERSION}"

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

install -d "${RELEASE_DIR}/bin"
install -d "${RELEASE_DIR}/web"

install -m 0755 "${ROOT_DIR}/build/manager-linux-amd64" "${RELEASE_DIR}/bin/manager"
rm -rf "${RELEASE_DIR}/web/dist"
cp -R "${ROOT_DIR}/web/dist" "${RELEASE_DIR}/web/"
cp -R "${ROOT_DIR}/scripts" "${RELEASE_DIR}/"

ln -sfn "${RELEASE_DIR}" "${INSTALL_ROOT}/current"
"${SYSTEMCTL_BIN}" restart "${SERVICE_NAME}"

echo "$(translate update_release_note): ${RELEASE_DIR}"
echo "$(translate update_current_note): ${INSTALL_ROOT}/current"
echo "$(translate update_restart_note): ${SERVICE_NAME}"
