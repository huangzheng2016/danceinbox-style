# Build and Deploy

The scaffold ships the files: `build.sh`, `release.sh`, `Dockerfile`, `.dockerignore`, `docker-compose.yml`, `Caddyfile`, `.env.example`, `script/mysql-init.sql`, `.github/workflows/ci.yml`. This page explains how they fit together.

## Local

```bash
cp .env.example .env                        # full profile only
docker compose up -d mysql redis            # full profile only
cp config/local.example.json config/local.json   # set jwt.secret
./build.sh
PASSWORD=... ./output/run.sh adduser admin ADMIN
./output/run.sh                             # http://127.0.0.1:8080
```

For frontend work run `pnpm dev` in `frontend/` (Vite proxies `/api` and `/papi` to :8080) next to `go run .`.

`build.sh` builds `frontend/dist` and then the Go binary that embeds it, skipping either half when its inputs have not changed. It reads `VERSION` (or a `vX.Y.Z` tag in CI) and never writes it.

## Versions

`VERSION` holds `MAJOR.MINOR.PATCH`. Only `./release.sh prod` changes it (patch + 1); commit the bump with the release. Minor and major bumps are edited by hand. The version is stamped into the binary and reported by `/healthz`.

## Release

```bash
./release.sh prod root@host     # or: ./release.sh test root@host
```

It rsyncs the source to `REMOTE_DIR` (default `/srv/<module>`), builds the image on the server, and recreates the one service. Real `config/*.json`, `.env`, and `data/` never leave the machine they are on: the rsync rules exclude them, and `--delete` does not touch them on the server.

Services: `app` (prod) and `app-test` share the image and differ by config. Uploaded files live in `upload.dir`; mount it as a volume (`./data/<env>/upload:/app/data/upload`) or they disappear with the container. Caddy terminates TLS (ACME, state in `data/caddy`) and is the only container with public ports; MySQL and Redis listen on `127.0.0.1` only.

## First deploy

On the server, before the first release:

```bash
mkdir -p /srv/<module>/config && cd /srv/<module>
# after one rsync (or scp) brings the examples:
cp .env.example .env && chmod 600 .env                     # real passwords
cp config/prod.example.json config/prod.json               # same passwords, real jwt.secret
cp config/test.example.json config/test.json
chmod 600 config/*.json
```

Create the config files before the first `docker compose up`. If the mount source is missing, Docker creates an empty directory in its place and the app exits at config load; remove that directory, then copy the file.

In containers the app reaches MySQL and Redis by service name (`mysql:3306`, `redis:6379`), not `127.0.0.1`. The example configs already do this.

Then create the first account: `docker compose run --rm app -c prod adduser admin ADMIN` with `PASSWORD` set in the environment (`-e PASSWORD=...`).

## Lite profile

For single-binary tools and small internal services with no separate database or cache:

- Copy `backend/scaffold-lite/` over the scaffold. It replaces `biz/dal/db/db.go` (SQLite via `github.com/glebarez/sqlite`, pure Go, no CGO), `biz/extension/cache/provider.go` (in-process cache), the example configs (a file path as `db.dsn`, no `redis` block), and `docker-compose.yml` (app + Caddy, SQLite under `data/<env>`).
- Delete `script/mysql-init.sql` and `.env.example`.
- Dependencies: `glebarez/sqlite` instead of `gorm.io/driver/mysql` and `go-redis`.
- Services, handlers, IDL, and the frontend are identical, so moving to the full profile later means swapping those files back and migrating the data.

The in-process cache does not survive restarts and is not shared between processes; the lite profile runs exactly one replica.

## CI

`.github/workflows/ci.yml` runs `pnpm check`, regenerates hz / gorm-gen / wire output and fails on any diff, then `gofmt`, `go vet`, and `go test`. `frontend/dist/.gitkeep` (copied from `frontend/public`) keeps the `//go:embed` valid on a clean checkout.

## Troubleshooting

- Certificates never issue: ports 80 and 443 (TCP and UDP) must be open to the internet.
- A rebuilt image does not run: `release.sh` already uses `--force-recreate`; when running compose by hand, do the same.
- `/healthz` reports `db: down` right after the first `up`: MySQL is still initializing; it reports `up` once ready.
