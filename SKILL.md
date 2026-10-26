---
name: danceinbox-style
description: "Use when creating a new Hertz (Go) + Vue 3 web app or admin console, or when adding features, endpoints, or pages to a project created from this skill (its CLAUDE.md names danceinbox-style). Ships a working scaffold: Thrift IDL + hz, gorm/gen, Wire, cookie JWT auth with roles, an embedded Vue 3 SPA with a no-library design system (shell, list/detail pages, dialogs, tables, toasts), tests, CI, and Docker/Caddy deploy."
---

# danceinbox-style

A full-stack scaffold that is real, runnable code: copy it, rename it, and it builds, serves, logs in, and manages an example resource end to end, including streamed agent runs with tool calls, file attachments, a CSV import wizard, and charts. New features are built by copying the example feature and the reference pages, so every endpoint and every page in a project looks and behaves the same.

Prefer copying over composing from rules. When the skill and your instinct disagree on how something looks or works, the reference code wins; if the reference code is wrong, fix it here in the skill so every project gets the fix.

`$SKILL` below is this skill's directory.

## Contents

| Path | What it is |
|---|---|
| `backend/scaffold/` | The project root: Go backend, IDL, configs, build/release/Docker/CI files, project `CLAUDE.md` |
| `backend/scaffold-lite/` | Files that replace their scaffold counterparts for the lite profile (SQLite, in-process cache) |
| `frontend/scaffold/` | The project's `frontend/`: Vite/TS/lint config, API client, auth store, router, reference pages, tests |
| `frontend/base/` | Shared components and composables, copied verbatim to `frontend/src/base/` |
| `frontend/styles/` | `tokens.css` + `base.css`, copied verbatim to `frontend/src/styles/` |
| `frontend/assets/fonts/` | Self-hosted fonts and licenses, copied to `frontend/public/fonts/` |
| `backend/workflow.md` | Layout, adding a feature, IDL and handler conventions, auth, testing, version pins |
| `backend/data.md` | Query, write, transaction, delete, migration, and cache rules |
| `backend/deploy.md` | Local run, versions, release, first deploy, lite profile, CI |
| `frontend/design-system.md` | Page patterns, which control to use, interaction rules, copy, tokens, review checklist |

## New project

Ask the user two things, then proceed without further questions:

1. **Module name** (e.g. `acme_portal`), used for the directory, Go module, and database.
2. **Profile.** *Full*: MySQL 8 + Redis, for multi-user production services. *Lite*: one binary with SQLite and an in-process cache, for internal tools and small services. Default to full when unsure.

Then, from an empty directory named after the module:

```bash
# 1. backend and project files
cp -r $SKILL/backend/scaffold/. .
# lite only:
cp -r $SKILL/backend/scaffold-lite/. . && rm script/mysql-init.sql .env.example

# 2. frontend
mkdir -p frontend/src/base frontend/src/styles frontend/public/fonts
cp -r $SKILL/frontend/scaffold/. frontend/
cp $SKILL/frontend/base/* frontend/src/base/
cp $SKILL/frontend/styles/*.css frontend/src/styles/
cp $SKILL/frontend/assets/fonts/* frontend/public/fonts/
mkdir -p frontend/dist && touch frontend/dist/.gitkeep

# 3. names
grep -rl MODULE_NAME . | xargs sed -i 's/MODULE_NAME/<module>/g'
sed -i 's/<module_name>/<module>/' frontend/package.json
sed -i 's/PROFILE/<full|lite>/' CLAUDE.md
#    then set app.name and app.storagePrefix in frontend/src/base/app.ts

# 4. Go dependencies and code generation
go mod init <module>
go get github.com/cloudwego/hertz github.com/hertz-contrib/sse gorm.io/gorm gorm.io/gen \
    github.com/golang-jwt/jwt/v5 github.com/google/wire github.com/google/uuid \
    golang.org/x/crypto golang.org/x/time github.com/glebarez/sqlite
go get gorm.io/driver/mysql github.com/redis/go-redis/v9     # full profile only
go get -tool github.com/google/wire/cmd/wire
go get gorm.io/plugin/dbresolver@v1.6.2
go install github.com/cloudwego/hertz/cmd/hz@v0.9.7
go install github.com/cloudwego/thriftgo@v0.4.1
./script/hz_update.sh
go get github.com/apache/thrift@v0.13.0
go run ./cmd/generate
go generate ./container
go mod tidy

# 5. verify
(cd frontend && pnpm install && pnpm check)
go vet ./... && go test ./...
cp config/local.example.json config/local.json       # set a 32+ char jwt.secret
cp .env.example .env && docker compose up -d mysql redis   # full profile only
./build.sh
PASSWORD=<pw> ./output/run.sh adduser admin ADMIN
./output/run.sh       # open http://127.0.0.1:8080, log in, create and delete an example
```

Every step must succeed before the next. If a tool fails, fix the tool or the input; never hand-write generated code to get past it. The version pins exist because those exact breakages happen (see `backend/workflow.md`).

Commit as `chore: scaffold`. Keep the `example` feature until the first real feature has been built from it, then delete it everywhere: model, service, IDL, handler, `convert.go` entries, frontend API, pages, route, nav, labels.

## Working in a generated project

The project's `CLAUDE.md` holds the day-to-day rules. Deeper references:

- New endpoint or feature: `backend/workflow.md` "Adding a feature".
- Anything touching queries, writes, or the cache: `backend/data.md`.
- New page: pick the reference page in `frontend/design-system.md` "Page patterns", copy it, then adapt.
- A shared component behaves wrong or is missing: fix or add it in `$SKILL/frontend/base/`, then re-copy `base/` into the project. Never fork a base component inside a project.

## Rules code cannot enforce

These are the decisions that keep projects consistent. Everything else is in the reference code.

**Backend**
- IDL first: every `/api` and `/papi` route is declared in `idl/` and generated by hz.
- One error vocabulary (`biz/errs`) and one envelope (`resp.Bind` / `resp.Fail`); messages are Chinese and meant for the end user.
- Typed queries only; state changes are compare-and-set in `Where`.
- Default-deny auth: sessions are required everywhere except `publicRoutes`.
- `go test ./...` runs with no external services; every bug fix starts with a failing test.

**Frontend**
- `base/` and `styles/` are used as-is. No component library, no CSS framework, no page-level restyling of shared classes or native controls.
- Pages are built from the reference pages; the shell owns the title.
- One primary button per area, destructive actions behind a confirm, feedback after every mutation, state in the URL.

**Code**
- Comments explain why and only where the code cannot. No comments that point at docs or describe history, no commented-out code, no "reserved for future" hooks or options, no compatibility shims in new code.
- Missing config fails at startup instead of falling back to a default. No fallback branches for states that cannot occur.

## Stack

Go: Hertz (+ hertz-contrib/sse for streams), Thrift IDL + hz, gorm + gorm/gen, Wire, golang-jwt, MySQL 8 + go-redis (full) or glebarez/sqlite (lite). Frontend: Vue 3 `<script setup>` + TypeScript, Vite, vue-router (hash history), Pinia, axios, markdown-it, ECharts, Vitest, ESLint, Prettier; no UI component library. Deploy: Docker Compose, Caddy (TLS), rsync release. Package managers: Go modules, pnpm via corepack.

Use these. A different framework, ORM, router mode, UI library, or i18n setup is a project-level decision the user makes explicitly, not something to switch to while building a feature.
