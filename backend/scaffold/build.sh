#!/bin/bash
# Builds frontend/dist and output/bin/MODULE_NAME. Each half is skipped when its
# inputs have not changed since the last build. VERSION is read, never written;
# release.sh owns version bumps.
set -euo pipefail
cd "$(dirname "$0")"

NAME=MODULE_NAME
VERSION="${GITHUB_REF_NAME:-}"; VERSION="${VERSION#v}"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || VERSION="$(tr -d '[:space:]' < VERSION)"
mkdir -p .cache output/bin

hash_of() {
    find "$@" -type f -not -path '*/node_modules/*' -not -path '*/dist/*' -print0 2>/dev/null |
        sort -z | xargs -0 sha1sum | sha1sum | cut -c1-40
}
changed() {  # changed <stamp-file> <hash>
    [ ! -f ".cache/$1" ] || [ "$(cat ".cache/$1")" != "$2" ]
}

FE_HASH="$(hash_of frontend/src frontend/public frontend/index.html frontend/package.json frontend/pnpm-lock.yaml frontend/vite.config.ts)"
if changed frontend "$FE_HASH" || [ ! -f frontend/dist/index.html ]; then
    echo "=== 前端构建 ==="
    (cd frontend && pnpm install --frozen-lockfile && pnpm build)
    echo "$FE_HASH" > .cache/frontend
else
    echo "=== 前端无变更,跳过 ==="
fi

GO_HASH="$(hash_of ./*.go go.mod go.sum biz cmd config container frontend/dist)-$VERSION-$(go env GOOS GOARCH | tr '\n' -)"
if changed go "$GO_HASH" || [ ! -f "output/bin/$NAME" ]; then
    echo "=== Go 构建 ==="
    go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "output/bin/$NAME" .
    echo "$GO_HASH" > .cache/go
else
    echo "=== Go 无变更,跳过 ==="
fi
cp script/run.sh output/
echo "版本 $VERSION: output/bin/$NAME"
