# Repository Guidelines

## Project Structure & Module Organization
`cmd/manager/` contains the Go entrypoint. `internal/` contains config, auth, HTTP API, SQLite access, runtime sync, and supporting packages. `web/` contains the React/Vite frontend source and build output. `scripts/` holds deployment helpers such as `bootstrap-reality-env.sh`, `check.sh`, `cleanup.sh`, and `install.sh`. Runtime data is stored in `data/` and generated Xray config in `generated/`; treat both as local state, not source.

## Build, Test, and Development Commands
- `go test ./...`: run backend tests.
- `go run ./cmd/manager`: start the backend locally.
- `cd web && npx tsc --noEmit && npm run build`: typecheck and build the frontend.
- `./scripts/bootstrap-reality-env.sh`: generate `XRAY_PRIVATE_KEY`, `XRAY_PUBLIC_KEY`, and `SESSION_SECRET`.
- `./scripts/check.sh && ./scripts/install.sh`: validate and install the host-level deployment.

## Deployment Artifact Requirement
Deployment uploads must include both prebuilt artifacts:

- `web/dist/`
- `build/manager-linux-amd64`

The server-side deployment flow assumes these two artifacts are already present in the project tree. The intended deployment path is to upload the full project with both artifacts included, then run:

```bash
./scripts/check.sh
./scripts/install.sh
```

Do not rely on the target server to build missing frontend or backend artifacts during deployment.

## Coding Style & Naming Conventions
Use the existing Go and TypeScript style in the repo. Prefer small responsibility-based packages/modules, `camelCase` for JS/TS identifiers, and `UPPER_SNAKE_CASE` for environment variables. Keep runtime, auth, store, and HTTP responsibilities separated as they are now.

## Testing Guidelines
Run Go tests plus frontend build/typecheck for changes: `go test ./...` and `cd web && npx tsc --noEmit && npm run build`. For deployment/runtime work, validate the host-script path with `./scripts/check.sh` and `./scripts/install.sh` on a target machine. When adding tests, keep them close to the Go packages under `internal/`.

## Commit & Pull Request Guidelines
This repository currently has no commit history, so there is no established convention to infer. Use Conventional Commit style going forward, for example `feat: add client traffic refresh` or `fix: handle xray service status`. PRs should include a short summary, touched areas, config or env changes, manual verification steps, and screenshots when `web/` UI changes.

## Security & Configuration Tips
Do not commit `.env`, `data/`, or `generated/`. Keep production secrets in environment variables, and verify `PANEL_BASE_URL`, `LINE_DOMAIN`, and `LINE_SERVER_ADDRESS` together before deploying.
