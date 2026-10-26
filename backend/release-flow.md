# Release Flow Reference

Concrete templates for `build.sh`, `release.sh`, `Dockerfile`, `Caddyfile`, `docker-compose.yml`, and the VERSION semantics. Drop these in at project birth and they keep working for the life of the repo.

## VERSION

A single file at the repo root with one line, `MAJOR.MINOR.PATCH`. Starts at:
```
1.0.0
```

Patch is auto-bumped by `build.sh` when sources changed, and by `release.sh prod` before remote deploy. `test` / `pre` deploys bump VERSION only for the remote build args; the local file is unchanged.

Always include VERSION in the same commit when it changed. Bumped VERSION + binary mismatch is a debugging nightmare.

## build.sh

Local-incremental build. Skips frontend/Go rebuilds when content hash unchanged.

```bash
#!/bin/bash
set -e

BUILD_VERSION=""
REQUESTED=""
while [ $# -gt 0 ]; do
    case "$1" in
        --version) BUILD_VERSION="${2:-}"; REQUESTED=1; shift ;;
        -h|--help) echo "用法: $0 [--version <version>]"; exit 0 ;;
        *) echo "未知参数: $1"; exit 1 ;;
    esac
    shift
done

RUN_NAME="<MODULE_NAME>"
TARGET_GOOS="${GOOS:-$(go env GOOS)}"
TARGET_GOARCH="${GOARCH:-$(go env GOARCH)}"
RUN_OUTPUT_NAME="$RUN_NAME"
[ "$TARGET_GOOS" = "windows" ] && RUN_OUTPUT_NAME="${RUN_OUTPUT_NAME}.exe"

HASH_DIR=".cache"
mkdir -p "$HASH_DIR"

read_version_file() { [ -f "$1" ] && tr -d '[:space:]' < "$1" || echo "1.0.0"; }
validate_version() { [[ "$1" =~ ^[0-9]+[.][0-9]+[.][0-9]+$ ]]; }
bump_patch() {
    IFS=. read -r maj min pat <<< "$1"
    echo "${maj}.${min}.$((pat + 1))"
}

resolve_build_version() {
    local file="VERSION" next write=1
    if [ -n "${BUILD_VERSION:-}" ]; then
        next="${BUILD_VERSION#v}"
        [ "${APP_SKIP_VERSION_BUMP:-0}" = "1" ] && write=0
    elif [ "${APP_SKIP_VERSION_BUMP:-0}" = "1" ]; then
        next="$(read_version_file "$file")"; write=0
    elif [ -n "${GITHUB_REF_NAME:-}" ] && [[ "${GITHUB_REF_NAME#v}" =~ ^[0-9]+[.][0-9]+[.][0-9]+$ ]]; then
        next="${GITHUB_REF_NAME#v}"; write=0
    else
        next="$(read_version_file "$file")"; write=0
    fi
    validate_version "$next" || { echo "VERSION 必须是 MAJOR.MINOR.PATCH" >&2; exit 1; }
    [ "$write" = "1" ] && printf "%s\n" "$next" > "$file"
    echo "$next"
}

BUILD_VERSION="$(resolve_build_version)"
GO_LDFLAGS="-s -w -X main.releaseMode=true -X main.version=${BUILD_VERSION}"

calc_hash() {
    find "$@" -type f \
        -not -path "*/node_modules/*" -not -path "*/dist/*" -not -path "*/.cache/*" \
        -print0 2>/dev/null | sort -z | xargs -0 md5sum 2>/dev/null | md5sum | cut -d' ' -f1
}

calc_go_hash() {
    (
        find . -maxdepth 1 -name "*.go" -print0 2>/dev/null | sort -z | xargs -0 md5sum 2>/dev/null
        find biz config container localize utils cmd script -type f -print0 2>/dev/null | sort -z | xargs -0 md5sum 2>/dev/null
        md5sum go.mod go.sum 2>/dev/null
        echo "frontend=${FRONTEND_HASH};ldflags=${GO_LDFLAGS};goos=${TARGET_GOOS};goarch=${TARGET_GOARCH}"
    ) | md5sum | cut -d' ' -f1
}

if ! command -v pnpm >/dev/null 2>&1; then
    command -v corepack >/dev/null 2>&1 || { echo "需要 pnpm 或 corepack"; exit 1; }
    # Pin the version — keep in sync with the Dockerfile's corepack prepare line.
    corepack enable && corepack prepare pnpm@10.0.0 --activate
fi

echo "=== 预取 Go 模块 ==="
go mod download

FRONTEND_HASH="$(calc_hash frontend/src frontend/public frontend/index.html frontend/admin.html frontend/package.json frontend/pnpm-lock.yaml frontend/vite.config.ts frontend/tsconfig.json)"
FRONTEND_HASH_FILE="$HASH_DIR/frontend.hash"
PREV_FRONTEND_HASH="$( [ -f "$FRONTEND_HASH_FILE" ] && cat "$FRONTEND_HASH_FILE" || true )"

FRONTEND_REBUILT=false
if [ "$FRONTEND_HASH" != "$PREV_FRONTEND_HASH" ] || [ ! -d "frontend/dist" ]; then
    echo "=== 构建前端 ==="
    (cd frontend && \
        { [ -d node_modules ] || pnpm install; } && \
        pnpm run build)
    [ -d "frontend/dist" ] || { echo "前端构建失败"; exit 1; }
    echo "$FRONTEND_HASH" > "$FRONTEND_HASH_FILE"
    FRONTEND_REBUILT=true
else
    echo "=== 跳过前端构建（无变更）==="
fi

# OPTIONAL: compressed embed hook.
# If you enabled the zstd-archive embed mode (see embed-strategy.md),
# uncomment the following lines so frontend/dist.tar.zst is refreshed before go build:
#   if [ "$FRONTEND_REBUILT" = true ] || [ ! -f "frontend/dist.tar.zst" ]; then
#       prepare_frontend_archive
#   fi
# `prepare_frontend_archive` is defined in embed-strategy.md.

GO_HASH="$(calc_go_hash)"
GO_HASH_FILE="$HASH_DIR/go.${TARGET_GOOS}.${TARGET_GOARCH}.hash"
PREV_GO_HASH="$( [ -f "$GO_HASH_FILE" ] && cat "$GO_HASH_FILE" || true )"

NEED_GO_BUILD=false
if [ "$GO_HASH" != "$PREV_GO_HASH" ]; then NEED_GO_BUILD=true; echo "=== 构建 Go（Go 变更）==="
elif [ "$FRONTEND_REBUILT" = true ]; then NEED_GO_BUILD=true; echo "=== 构建 Go（前端变更，需重新 embed）==="
elif [ ! -f "output/bin/${RUN_OUTPUT_NAME}" ]; then NEED_GO_BUILD=true; echo "=== 构建 Go（二进制不存在）==="
fi

if [ "$NEED_GO_BUILD" = true ]; then
    if [ "${APP_SKIP_VERSION_BUMP:-0}" != "1" ] && [ -z "$REQUESTED" ] && \
       { [ -z "${GITHUB_REF_NAME:-}" ] || ! [[ "${GITHUB_REF_NAME#v}" =~ ^[0-9]+[.][0-9]+[.][0-9]+$ ]]; }; then
        BUILD_VERSION="$(bump_patch "$BUILD_VERSION")"
        printf "%s\n" "$BUILD_VERSION" > VERSION
        GO_LDFLAGS="-s -w -X main.releaseMode=true -X main.version=${BUILD_VERSION}"
    fi
    mkdir -p output/bin
    cp script/* output 2>/dev/null || true
    chmod +x output/bootstrap.sh 2>/dev/null || true
    go build -ldflags "${GO_LDFLAGS}" -o "output/bin/${RUN_OUTPUT_NAME}" .
    # Recompute AFTER the bump: the stored hash must match the binary actually
    # produced (post-bump ldflags), or every run rebuilds and re-bumps forever.
    GO_HASH="$(calc_go_hash)"
    echo "$GO_HASH" > "$GO_HASH_FILE"
else
    echo "=== 跳过 Go 构建（无变更）==="
fi

echo "=== 构建完成 ==="
echo "版本: ${BUILD_VERSION}"
echo "二进制: output/bin/${RUN_OUTPUT_NAME}"
```

## script/bootstrap.sh

Sits next to the compiled binary in `output/`. Wraps the `-c` flag.

```bash
#!/bin/bash
set -euo pipefail
# Do NOT cd into the script dir. Config loads from ./config relative to the
# CALLER's cwd (repo root in dev); the binary is invoked by path instead.
exec "$(dirname "$0")/bin/<MODULE_NAME>" -c "${APP_CONFIG:-local}" "$@"
```

`./output/bootstrap.sh` runs the local config by default. In Docker the entrypoint is `/app/<MODULE_NAME> -c prod` (see Dockerfile).

## release.sh

```bash
#!/bin/bash
set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  ./release.sh [prod|test|pre] [remote]
Env:
  SSH_PORT=22
  REMOTE_DIR=/root/<MODULE_NAME>
  NO_CACHE=1                    docker compose build --no-cache
  NO_RESTART=1                  build only
  APP_SKIP_RELEASE_BUMP=1   skip patch bump
  INFRA_SERVICES="mysql redis"  infra services ensured up before recreate;
                                set INFRA_SERVICES= (empty) for lightweight profile
EOF
}

bump_patch() {
    [[ "$1" =~ ^[0-9]+[.][0-9]+[.][0-9]+$ ]] || { echo "bad VERSION $1" >&2; exit 1; }
    IFS=. read -r maj min pat <<< "$1"
    echo "${maj}.${min}.$((pat + 1))"
}

TARGET="${1:-prod}"
if [[ "${TARGET}" == *@* ]]; then REMOTE="$TARGET"; TARGET="prod"; else REMOTE="${2:-${REMOTE:-root@your.host}}"; fi
REMOTE_PORT="${SSH_PORT:-22}"
REMOTE_DIR="${REMOTE_DIR:-/root/<MODULE_NAME>}"
NO_RESTART="${NO_RESTART:-}"
NO_CACHE="${NO_CACHE:-}"
INFRA_SERVICES="${INFRA_SERVICES-mysql redis}"

[[ "$TARGET" =~ ^(-h|--help|help)$ ]] && { usage; exit 0; }

BASE="$(tr -d '[:space:]' < VERSION)"
VERSION="$BASE"
case "$TARGET" in
    prod) SERVICE="app";       CONFIG="prod"; [ "${APP_SKIP_RELEASE_BUMP:-0}" != "1" ] && { VERSION="$(bump_patch "$BASE")"; printf '%s\n' "$VERSION" > VERSION; } ;;
    test) SERVICE="app-test";  CONFIG="test"; [ "${APP_SKIP_RELEASE_BUMP:-0}" != "1" ] && VERSION="$(bump_patch "$BASE")" ;;
    pre)  SERVICE="app-pre";   CONFIG="pre";  [ "${APP_SKIP_RELEASE_BUMP:-0}" != "1" ] && VERSION="$(bump_patch "$BASE")" ;;
    *) usage >&2; exit 1 ;;
esac

printf -v VERSION_SHELL "%q" "$VERSION"

echo "=== Deploy ==="
echo "环境: $CONFIG | 服务: $SERVICE | VERSION: $VERSION | 远端: $REMOTE:$REMOTE_DIR"

# Rule order is load-bearing (rsync is first-match): the example templates hit
# --include and travel; every other config/*.json is excluded on the command
# line, which also shields server-side hand-made configs from --delete (why the
# gitignore filter alone cannot do this: see ".gitignore").
rsync -avz --delete \
    --include='config/*.example.json' \
    --exclude='config/*.json' \
    --filter=':- .gitignore' \
    --exclude='.git' --exclude='.idea' --exclude='.vscode' --exclude='.claude' \
    --exclude='.cache' --exclude='output' --exclude='data' \
    --exclude='frontend/dist' --exclude='frontend/node_modules' \
    --exclude='*.log' --exclude='.env' --exclude='.env.local' \
    -e "ssh -p ${REMOTE_PORT}" ./ "${REMOTE}:${REMOTE_DIR}/"

BUILD_FLAGS=""
[ -n "$NO_CACHE" ] && BUILD_FLAGS="--no-cache"

ssh "${REMOTE}" -p "${REMOTE_PORT}" \
  "cd ${REMOTE_DIR} && VERSION=${VERSION_SHELL} docker compose build ${BUILD_FLAGS} ${SERVICE}"

if [ -z "$NO_RESTART" ]; then
    INFRA_CMD="true"
    [ -n "$INFRA_SERVICES" ] && INFRA_CMD="docker compose up -d --remove-orphans ${INFRA_SERVICES}"
    ssh "${REMOTE}" -p "${REMOTE_PORT}" \
      "cd ${REMOTE_DIR} && docker compose run --rm --no-deps caddy caddy validate --config /etc/caddy/Caddyfile && \
       ${INFRA_CMD} && \
       docker compose up -d --no-deps --force-recreate ${SERVICE} && \
       docker compose up -d --force-recreate --remove-orphans caddy"
fi

ssh "${REMOTE}" -p "${REMOTE_PORT}" "cd ${REMOTE_DIR} && docker compose images ${SERVICE} caddy"
```

Replace `your.host` with the actual target hostname (or always pass `REMOTE=...`).

## .gitignore

`release.sh` rsyncs with `--filter=':- .gitignore'`, so this file doubles as the upload filter — anything listed here is neither committed nor sent to the server (one deliberate exception: the committed example configs, see below):

```gitignore
output/
frontend/dist
frontend/node_modules
data/
.cache/
*.log
.env
.env.local
config/*.json
!config/*.example.json
```

The `!config/*.example.json` negation is load-bearing for GIT only: real per-environment configs carry secrets and must be ignored, while the templates must stay committed. Rsync's dir-merge filter does NOT honor gitignore `!` negations (a `!` line is read as a literal pattern), so the templates travel only via the explicit `--include` `release.sh` passes BEFORE the `--filter` (first-match) — and the paired command-line `--exclude='config/*.json'`, not the gitignore, is what shields hand-made server-side configs from `rsync --delete` (see "First Deploy").

## First Deploy: create the server-side config BEFORE the first release

Real `config/<env>.json` files are gitignored AND rsync-excluded — only the `*.example.json` templates travel (see ".gitignore"). On a fresh server the bind-mount source is missing, and Docker's short-syntax mount auto-creates it as an empty DIRECTORY: the app exits fatal on config load and the container crash-loops. Pre-create all three environments' configs by hand:

```bash
ssh root@your.host
mkdir -p /root/<MODULE_NAME>/config
# rsync once without building so the *.example.json templates land:
#   ./release.sh prod root@your.host   # let rsync run, then Ctrl-C before the build — or just scp the examples
cd /root/<MODULE_NAME>/config
# A previous failed `up` may have left empty DIRECTORIES named prod.json/ etc.
# cp into them does NOT fail — it silently nests the copy inside. Remove first:
for env in prod test pre; do
    if [ -d "${env}.json" ]; then
        ls -la "${env}.json"          # expect EMPTY; check for nested secrets before deleting
        rm -rf "${env}.json"
    fi
done
for env in prod test pre; do
    cp "${env}.example.json" "${env}.json"
    # now edit ${env}.json: real DSNs, jwt.secret, cookie_secure, etc.
done
chmod 600 prod.json test.json pre.json
```

Create all three up front even if you only deploy `prod` for now — the first `release.sh test`/`pre` hits the same trap on its own config. Ongoing releases are safe: the command-line `--exclude` keeps `--delete` away from the server-side configs; edit them on the server only.

## Dockerfile

```dockerfile
# syntax=docker/dockerfile:1

# Stage 1: frontend
FROM node:24-slim AS frontend-builder
RUN corepack enable && corepack prepare pnpm@10.0.0 --activate
WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store,sharing=locked \
    pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm run build

# Stage 2: Go
FROM golang:1.26-bookworm AS go-builder
ENV GOPROXY=https://goproxy.cn,direct
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked go mod download
COPY . .
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,sharing=locked \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w -X main.releaseMode=true -X main.version=${VERSION}" \
        -o /out/<MODULE_NAME> .

# Stage 3: runtime
FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends tzdata ca-certificates && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=go-builder /out/<MODULE_NAME> /app/<MODULE_NAME>
RUN chmod +x /app/<MODULE_NAME>
ENV TZ=Asia/Shanghai
CMD ["/app/<MODULE_NAME>", "-c", "prod"]
```

## .dockerignore

`docker compose build` runs ON the server with build context `.` = `REMOTE_DIR` — which by then holds the hand-made `config/<env>.json`, the `.env` secrets, and the live `data/` tree (MySQL files, uploads, Caddy ACME state). Without a `.dockerignore` every build tars all of that to the daemon and Stage 2's `COPY . .` bakes it into the go-builder layers: slow builds, and secrets plus DB snapshots sitting in dangling intermediate images. Keep this file at the repo root — rsync carries it to the server like any committed file:

```dockerignore
.git
data/
output/
.cache/
*.log
.env
.env.local
config/*.json
!config/*.example.json
frontend/node_modules
frontend/dist
```

`frontend/dist` is safe to exclude in both embed modes — Stage 1 rebuilds it and Go gets it via `COPY --from=frontend-builder` (see embed-strategy.md). Unlike rsync's dir-merge filter (see ".gitignore"), Docker DOES honor `!` negations. The real `config/<env>.json` still reaches the app container as a runtime bind mount; it is never image content.

## Caddyfile

```
{
    email you@example.com
    storage file_system /data
}

(app_proxy) {
    encode zstd gzip
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Content-Type-Options "nosniff"
        Referrer-Policy "strict-origin-when-cross-origin"
    }
}

your-domain.tld {
    import app_proxy
    reverse_proxy app:8080
}

pre.your-domain.tld {
    import app_proxy
    reverse_proxy app-pre:8080
}

test.your-domain.tld {
    import app_proxy
    reverse_proxy app-test:8080
}
```

ACME state in `./data/caddy/`. Back this volume up.

## docker-compose.yml

```yaml
services:
    mysql:
        image: mysql:8
        container_name: mysql
        ports:
            - "127.0.0.1:3306:3306"
        environment:
            MYSQL_ROOT_PASSWORD: "${MYSQL_ROOT_PASSWORD:-change_me_root_password}"
            MYSQL_DATABASE: "<MODULE_NAME>"
            TZ: Asia/Shanghai
        volumes:
            - ./data/mysql:/var/lib/mysql
        command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
        restart: unless-stopped

    redis:
        image: redis:6
        container_name: redis
        ports:
            - "127.0.0.1:6379:6379"
        environment:
            REDIS_PASSWORD: "${REDIS_PASSWORD:-change_me_redis_password}"
        volumes:
            - ./data/redis:/data
        command: redis-server --requirepass ${REDIS_PASSWORD:-change_me_redis_password}
        restart: unless-stopped

    app:
        build:
            context: .
            args:
                VERSION: ${VERSION:-dev}
        container_name: app
        environment:
            TZ: Asia/Shanghai
        depends_on:
            - mysql
            - redis
        volumes:
            - ./config/prod.json:/app/config/prod.json:ro
            - ./data/upload:/app/data/upload
        command: ["/app/<MODULE_NAME>", "-c", "prod"]
        restart: unless-stopped

    app-pre:
        build:
            context: .
            args:
                VERSION: ${VERSION:-dev}
        container_name: app-pre
        environment:
            TZ: Asia/Shanghai
        depends_on:
            - mysql
            - redis
        volumes:
            - ./config/pre.json:/app/config/pre.json:ro
            - ./data/pre/upload:/app/data/upload
        command: ["/app/<MODULE_NAME>", "-c", "pre"]
        restart: unless-stopped

    app-test:
        build:
            context: .
            args:
                VERSION: ${VERSION:-dev}
        container_name: app-test
        environment:
            TZ: Asia/Shanghai
        depends_on:
            - mysql
            - redis
        volumes:
            - ./config/test.json:/app/config/test.json:ro
            - ./data/test/upload:/app/data/upload
        command: ["/app/<MODULE_NAME>", "-c", "test"]
        restart: unless-stopped

    caddy:
        image: caddy:2
        container_name: caddy
        ports:
            - 80:80
            - 443:443
            - 443:443/udp
        volumes:
            - ./Caddyfile:/etc/caddy/Caddyfile:ro
            - ./data/caddy/data:/data
            - ./data/caddy/config:/config
        restart: unless-stopped
```

Real credentials go in a server-side `.env` next to `docker-compose.yml`, never committed. `docker compose` picks `.env` up automatically.

Compose networking note for the hand-made `config/<env>.json` (see "First Deploy"): the app reaches MySQL/Redis over the compose network by SERVICE NAME, so `prod.json` (and `pre.json` / `test.json`) must set `mysql.host` to `"mysql"` and `redis.addr` to `"redis:6379"` — NOT `127.0.0.1`. Inside the app container, `127.0.0.1` is the container's own loopback; the `127.0.0.1:3306` / `127.0.0.1:6379` port publishes above only expose the services to the HOST, not to sibling containers. (`config/local.json` keeps `127.0.0.1` because a local smoke test runs the binary on the host.)

## CI (GitHub Actions example)

`.github/workflows/build.yml`:
```yaml
name: build

on:
  push:
    branches: [main, master]
    tags: ['v*']
  pull_request:

jobs:
  build:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - uses: pnpm/action-setup@v4
        with: { version: 10 }
      - uses: actions/setup-node@v4
        with: { node-version: 24, cache: pnpm, cache-dependency-path: frontend/pnpm-lock.yaml }
      - name: install frontend
        working-directory: frontend
        run: pnpm install --frozen-lockfile
      - name: check frontend
        working-directory: frontend
        run: pnpm check
      - name: verify generated sources
        run: |
          ./script/hz_update.sh
          go run ./cmd/generate
          go generate ./container
          git diff --exit-code -- biz/handler biz/router biz/model biz/dal/mysql/query container
      - name: go vet
        run: go vet ./...
      - name: gofmt check
        run: test -z "$(gofmt -l .)"
      - name: build
        env:
          APP_SKIP_VERSION_BUMP: '1'
        run: ./build.sh
      - name: go test
        run: go test ./...
```

The `APP_SKIP_VERSION_BUMP=1` keeps CI from churning the VERSION file. Real bumps come from `release.sh prod` on the maintainer's machine, or from a `v*` tag (`GITHUB_REF_NAME=vX.Y.Z` lets `build.sh` use the tag's version verbatim).

## VERSION Semantics Recap

| Trigger | Effect on local VERSION | Effect on built binary |
|--------|------------------------|------------------------|
| `./build.sh` with source changes | patch+1, written | new VERSION |
| `./build.sh` no source changes | unchanged | unchanged |
| `./build.sh --version 2.0.0` | unchanged unless APP_SKIP_VERSION_BUMP unset (then 2.0.0 written) | 2.0.0 |
| `GITHUB_REF_NAME=v1.2.3 ./build.sh` (CI tag) | unchanged | 1.2.3 |
| `APP_SKIP_VERSION_BUMP=1 ./build.sh` | unchanged | reads VERSION file as-is |
| `./release.sh prod` | patch+1, written; commit manually | patch+1 used for `docker compose build` |
| `./release.sh test` | unchanged | patch+1 used for remote build only |
| `./release.sh pre` | unchanged | patch+1 used for remote build only |

If your build pipeline ends with a dirty VERSION on a `prod` deploy, commit it.

## Pitfalls

- Caddy not picking up a new certificate after first boot: ensure 80 + 443 (TCP + UDP for HTTP/3) are open on the host. ACME challenges fail silently otherwise.
- Server can't read config / container crash-loops (`Restarting`) right after first deploy: the real `config/<env>.json` was never created on the server and Docker auto-created an empty DIRECTORY at the mount source. Fix per "First Deploy": `rm -rf` the stray directory (a `cp` into it nests silently), then create the file from `config/<env>.example.json`. Mandatory first-deploy step, not optional.
- `docker compose build` succeeds but the new code is not running: you forgot `--force-recreate`. Always run `up -d --force-recreate <service>` after build.
- After `release.sh test` (or `pre`), the remote image was built with a VERSION one patch higher than the local repo's VERSION file: that is by design. `test`/`pre` only compute the bumped value in memory and pass it as a remote `docker compose build` arg — nothing writes a VERSION file on either side. Only `prod` bumps and writes the local VERSION file.
- Permission denied on `data/` during first boot: `mkdir -p data/{mysql,redis,upload,test/upload,pre/upload,caddy/data,caddy/config}` on the server first, or run docker-compose as root.
