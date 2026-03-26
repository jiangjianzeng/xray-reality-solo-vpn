# xray-reality-solo-vpn

English | [简体中文](./README.zh-CN.md) | [日本語](./README.ja.md)

`xray-reality-solo-vpn` is a self-hosted secure access manager and control panel for a single VPS. `Solo VPN` is used as the UI/product name inside the panel.

## Upstream References

- Xray-core upstream repository: <https://github.com/XTLS/Xray-core>
- REALITY upstream repository: <https://github.com/XTLS/REALITY>

This project builds on `Xray-core + VLESS + REALITY`, but it is not a mirror of
those upstream repositories and it is not an official XTLS project. Refer to
the upstream repositories for protocol behavior, parameter semantics, and
compatibility details.

## Product Scope

- Single-machine deployment
- First-run admin setup and login
- Client create / disable / delete
- `vless://` share link generation
- Mihomo / Clash Meta subscription export
- Basic traffic visibility and last-seen state
- Xray runtime config regeneration and reload
- Admin password update inside the panel

This project is not a multi-node control plane, not a public shared-access platform, and not a multi-tenant distribution SaaS.

## Stack

- `Go` manager
- `React + Vite + Tailwind + shadcn/ui-style components` frontend
- `SQLite` embedded into the manager
- `Xray-core + VLESS + REALITY`
- `systemd + Caddy + Nginx stream` for host-level deployment

## Project Layout

- `cmd/manager/`
  Go entrypoint
- `internal/`
  Config, auth, HTTP API, SQLite access, subscriptions, runtime sync
- `web/`
  Frontend source and build output
- `scripts/install.sh`
  Interactive host installer
- `scripts/check.sh`
  Residue and port detection
- `scripts/cleanup.sh`
  Old deployment cleanup
- `deploy/`
  systemd, Caddy, and Nginx templates
- `scripts/bootstrap-reality-env.sh`
  Generates `XRAY_PRIVATE_KEY`, `XRAY_PUBLIC_KEY`, and `SESSION_SECRET`
- `generated/server.json`
  Generated Xray runtime config
- `data/manager.db`
  SQLite database

## Host Install

Run on the target Ubuntu VPS:

```bash
./scripts/check.sh
./scripts/install.sh
```

`install.sh` asks only for the required deployment values, writes `/etc/xray-reality-solo-vpn/app.env`, installs host services, and prints a one-time setup URL.

## Deployment Artifact Requirement

Uploads for server deployment must already include both prebuilt artifacts:

- `web/dist/`
- `build/manager-linux-amd64`

The intended deployment flow is:

1. Build the frontend and backend artifacts locally
2. Upload the full project directory with both artifacts included
3. Run:

```bash
./scripts/check.sh
./scripts/install.sh
```

Do not rely on the target server to build missing frontend or backend artifacts during deployment.

## Host Services

- `xray-reality-solo-vpn.service`
  Go API + static frontend host on `127.0.0.1:3000`
- `xray.service`
  Reality server on `127.0.0.1:2443`
- `caddy.service`
  panel HTTP/HTTPS handling
- `nginx.service`
  `443/tcp` SNI splitter

## Domains And Addresses

- `PANEL_DOMAIN`
  Panel domain, for example `panel.example.com`
- `LINE_DOMAIN`
  Logical line domain shown in UI and subscriptions
- `LINE_SERVER_ADDRESS`
  The real dial address for clients. When fake-IP DNS, TUN loops, or local proxy recursion may interfere, use the server public IP directly.

## Setup Flow

- Public `/setup` is locked by default
- Installer prints a one-time URL:
  `https://<panel-domain>/_/setup/<token>`
- Only visiting that URL can authorize first-run admin creation
- After setup succeeds, the URL expires permanently and login moves to `/login`

## Local Development

Backend:

```bash
go test ./...
go run ./cmd/manager
```

Frontend:

```bash
npm --prefix web install
npm --prefix web run dev
npm --prefix web run build
```

There is no Docker / Compose deployment path anymore. Deployment is host-script based through `scripts/install.sh` plus the templates in `deploy/`.

## Deployment Notes

- Keep `.env`, `generated/`, and `data/` out of Git
- `panel.example.com` and `line.example.com` should usually point to the same VPS
- If the line is unstable while the server is healthy, check DNS, fake-IP behavior, TUN loops, and VPS route quality before suspecting the panel
- For mainland China usage, route quality usually matters more than panel code

## License

This project is dual-licensed under either:

- MIT (`LICENSE-MIT`)
- Apache License 2.0 (`LICENSE-APACHE`)

You may use this project under either license, at your option.
