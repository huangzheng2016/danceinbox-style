#!/bin/bash
# ./release.sh prod|test [user@host]
# Uploads the source with rsync, builds the image on the server, recreates the
# service. prod bumps the patch version in VERSION; commit it afterwards.
set -euo pipefail
cd "$(dirname "$0")"

ENV="${1:-}"
REMOTE="${2:-${REMOTE:-root@your.host}}"
REMOTE_DIR="${REMOTE_DIR:-/srv/MODULE_NAME}"
SSH="ssh -p ${SSH_PORT:-22}"

case "$ENV" in
    prod) SERVICE=app ;;
    test) SERVICE=app-test ;;
    *) echo "用法: $0 prod|test [user@host]" >&2; exit 1 ;;
esac

VERSION="$(tr -d '[:space:]' < VERSION)"
if [ "$ENV" = prod ]; then
    IFS=. read -r MAJ MIN PAT <<< "$VERSION"
    VERSION="$MAJ.$MIN.$((PAT + 1))"
    echo "$VERSION" > VERSION
fi
echo "=== 发布 $ENV $VERSION 到 $REMOTE:$REMOTE_DIR ==="

# Order matters (first match wins): example configs travel, real configs and
# anything gitignored stay local, and --delete never touches server-only files.
rsync -az --delete \
    --include='config/*.example.json' --exclude='config/*.json' \
    --exclude='.git' --exclude='.env' --exclude='data/' \
    --filter=':- .gitignore' \
    -e "$SSH" ./ "$REMOTE:$REMOTE_DIR/"

$SSH "$REMOTE" "cd $REMOTE_DIR && \
    VERSION=$VERSION docker compose build $SERVICE && \
    docker compose up -d --no-deps --force-recreate $SERVICE && \
    docker compose up -d caddy && \
    docker compose ps"
