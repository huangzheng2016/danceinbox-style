#!/bin/bash
# Runs the "New project" commands from SKILL.md, verbatim, in a temp directory and
# checks the result: frontend gate, go vet/test, build, codegen drift, and (lite)
# a served /healthz. Run after every change to the skill.
#
#   scripts/verify.sh [lite|full]   (default lite; full skips the docker and serve steps)
#   E2E=1 scripts/verify.sh lite    also drives the app in Chromium (scripts/e2e)
#
# Needs go, pnpm (corepack), python3, curl. Downloads modules on first run.
set -euo pipefail

PROFILE="${1:-lite}"
SKILL="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)/acme"
mkdir -p "$WORK"
export PATH="$HOME/go/bin:$PATH"

python3 - "$SKILL/SKILL.md" "$PROFILE" > "$WORK/boot.sh" <<'EOF'
import sys
text, profile = open(sys.argv[1]).read(), sys.argv[2]
start = text.index("```bash\n# 1. backend") + len("```bash\n")
lines = text[start:text.index("```", start)].splitlines()
print("set -euo pipefail")
skip = False
for line in lines:
    if line.startswith("# lite only:"):
        skip = profile != "lite"
        continue
    if skip:
        skip = False
        continue
    if "# full profile only" in line and profile == "lite":
        continue
    line = line.replace("<module>", "acme").replace("<full|lite>", profile).replace("<pw>", "verify-password")
    if "docker compose" in line or (profile == "full" and "./output/run.sh" in line):
        print(f"echo 'skipped: {line.split('#')[0].strip()}'")
        continue
    if line.startswith("./output/run.sh "):
        print("(./output/run.sh > server.log 2>&1 &); sleep 2")
        print("curl -fsS 127.0.0.1:8080/healthz && echo && curl -fsS -o /dev/null 127.0.0.1:8080/")
        print("pkill -f output/bin/acme || true")
        continue
    print(line)
    if line.startswith("cp config/local.example.json"):
        print("sed -i 's/replace-with-at-least-32-random-characters/verify-secret-verify-secret-verify-xx/' config/local.json")
EOF

cd "$WORK"
echo "=== boot ($PROFILE) in $WORK ==="
SKILL="$SKILL" bash boot.sh

echo "=== codegen drift ==="
git init -q && git add -A && git -c user.name=v -c user.email=v@v commit -qm scaffold
./script/hz_update.sh >/dev/null && go run ./cmd/generate >/dev/null && go generate ./container
git diff --exit-code
test -z "$(gofmt -l .)"

if [ "${E2E:-}" = 1 ] && [ "$PROFILE" = lite ]; then
    echo "=== e2e ==="
    (cd "$SKILL/scripts/e2e" && pnpm install --silent && pnpm exec playwright install chromium >/dev/null)
    mkdir -p shots
    for suite in auth-and-crud list-and-detail runs-files-import; do
        rm -rf data
        for u in "admin ADMIN" "alice USER" "albert USER" "bob USER"; do
            set -- $u
            PASSWORD=password1 ./output/run.sh adduser "$1" "$2" >/dev/null
        done
        ./output/run.sh > "server-$suite.log" 2>&1 &
        server=$!
        sleep 2
        E2E_SHOTS="$WORK/shots" node "$SKILL/scripts/e2e/$suite.mjs" || { kill $server; exit 1; }
        kill $server
        wait $server 2>/dev/null || true
    done
fi

echo "=== ok: $PROFILE profile scaffold builds, tests, and runs ==="
