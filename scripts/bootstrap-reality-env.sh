#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

if [[ ! -f .env ]]; then
  echo "Missing .env"
  exit 1
fi

ensure_line() {
  local key="$1"
  local value="$2"

  if grep -Eq "^${key}=" .env; then
    if grep -Eq "^${key}=$" .env; then
      perl -0pi -e "s/^${key}=\\n/${key}=${value}\\n/m" .env
    fi
  else
    printf '%s=%s\n' "$key" "$value" >> .env
  fi
}

parse_x25519_output() {
  local key_output="$1"

  XRAY_PRIVATE_KEY="$(printf '%s\n' "$key_output" | awk -F': ' '/^Private key:/ {print $2}')"
  XRAY_PUBLIC_KEY="$(printf '%s\n' "$key_output" | awk -F': ' '/^Public key:/ {print $2}')"

  if [[ -z "${XRAY_PRIVATE_KEY}" || -z "${XRAY_PUBLIC_KEY}" ]]; then
    XRAY_PRIVATE_KEY="$(printf '%s\n' "$key_output" | awk -F': ' '/^PrivateKey:/ {print $2}')"
    XRAY_PUBLIC_KEY="$(printf '%s\n' "$key_output" | awk -F': ' '/^Password:/ {print $2}')"
  fi
}

if ! grep -Eq '^XRAY_PRIVATE_KEY=.+$' .env || ! grep -Eq '^XRAY_PUBLIC_KEY=.+$' .env; then
  if ! command -v xray >/dev/null 2>&1; then
    echo "Missing xray binary. Please install Xray first or run this script on a host with xray available."
    exit 1
  fi

  KEY_OUTPUT="$(xray x25519 | tr -d '\r')"
  parse_x25519_output "$KEY_OUTPUT"

  if [[ -z "${XRAY_PRIVATE_KEY}" || -z "${XRAY_PUBLIC_KEY}" ]]; then
    echo "Failed to parse X25519 key output."
    printf '%s\n' "$KEY_OUTPUT"
    exit 1
  fi

  ensure_line "XRAY_PRIVATE_KEY" "$XRAY_PRIVATE_KEY"
  ensure_line "XRAY_PUBLIC_KEY" "$XRAY_PUBLIC_KEY"
fi

if ! grep -Eq '^SESSION_SECRET=.+$' .env; then
  ensure_line "SESSION_SECRET" "$(openssl rand -hex 32)"
fi
