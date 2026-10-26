---
name: danceinbox-style
description: "Use when (a) bootstrapping a new Hertz (Go) + Vue 3 full-stack project from scratch, or the user references our scaffold / Hertz+Vue starter / asks how to lay out a new web service; or (b) building Vue 3 frontend pages/components in any project that uses this design system. One skill, two parts: Part 1 locks the project skeleton (Thrift IDL + hz codegen + Wire + GORM, auth, CI/CD, testing, output discipline); Part 2 locks the component-library-free visual system (tokens.css + base.css global styles + shared components, flat paper palette with a single cyan accent, fixed app shell, split layouts, Chinese ops-console copy, full dark theme)."
---

# danceinbox-style

One skill, two parts. **Part 1 (Scaffold)** is the Hertz (Go) + Vue 3 full-stack project skeleton: Thrift IDL + `hz` codegen + Wire DI + GORM + JWT for backend, Vue 3 + Vite + Pinia + axios + VueUse for frontend, Docker Compose + Caddy + `release.sh` for deploy. **Part 2 (Style)** is the visual system those projects use: no component library, native controls styled by a global `base.css`, flat paper palette with a single cyan accent, fixed app shell, split layouts, Chinese ops-console copy.

## When to Use

Use when you are about to:
- Create a NEW Hertz + Vue 3 project from zero, or the user says "our Hertz+Vue scaffold", "the Hertz+Vue starter", "lay out a new web service" (Part 1).
- Make a decision about project layout, dependencies, IDL workflow, CI, or test conventions in a project this skill seeded (Part 1, on every feature, not just day one).
- Write or modify a Vue 3 page/component in any project seeded by this skill (Part 2).
- Any task where the user says "use our design system", "match the shared visual system", "no-component-library admin console" (Part 2).

Do NOT use for:
- Projects that already have a different established stack or design system, non-Hertz backends, non-Vue frontends, or pure backend tasks with no scaffold or design-system question.

---

# Part 1 - Scaffold (project skeleton)

## The Iron Rule (Part 1)

**IDL first. Code generation is not optional, it is the contract.**

Every HTTP handler reachable through `/api/*` or `/papi/*` MUST originate in `idl/api.thrift` or `idl/papi.thrift`. Never hand-write a route under those prefixes, never edit `biz/router/**` (the per-route `middleware.go` mw hooks are the one editable file there), never edit the function signature of a generated handler. If you find yourself writing `r.POST("/api/foo", ...)` directly, **STOP - you are reverting to default-agent behavior**. Add the method to the IDL, re-run `hz update`, then fill in the handler body.

The only routes that bypass IDL are: WebSocket upgrades in `customizedRegister(r)`, root-level utilities (`/ping`, `/livez`, `/healthz`), static assets served from `frontend_embed.go`, and the SPA fallback. SSE keeps its IDL/hz route, generated request DTO, middleware, and policy entry; the generated handler writes the stream and bypasses gzip/ETag. Full manual-route table: `backend/idl-workflow.md`.

## Mandatory Boot Sequence

When creating the project from scratch, do these in order. Do not parallelize, do not skip:

1. **Resolve module name and folder.** Pick `MODULE_NAME` (e.g. `acme_portal`); folder name should match. Confirm with the user if ambiguous. Settle the infrastructure footprint now (see "Infrastructure Footprint Decision" below) - it decides the dependencies added in step 3.
2. **Scaffold root layout.** Create the directory tree per "Project Layout" in `backend/backend-skeleton.md`. `mkdir` only, no business code yet.
3. **`go mod init <MODULE_NAME>`** and resolve dependencies versionless per `backend/backend-skeleton.md` ("Dependencies" - `go get` the module set, `go mod tidy` AFTER source exists, never hand-pick versions except the listed cliffs). Install the codegen toolchain (`hz`, `wire`, `thriftgo`) per "Codegen Toolchain" below.
4. **Write `idl/base.thrift` + `idl/api.thrift` + `idl/papi.thrift`** with at least one trivial service each (e.g. `Ping`). See `backend/idl-workflow.md` for the canonical header and conventions.
5. **Run the first generation** (both commands, per `backend/idl-workflow.md`): `hz new -idl idl/api.thrift -mod <MODULE_NAME> -handler_dir biz/handler -model_dir biz/model`, then `hz update -idl idl/papi.thrift -handler_dir biz/handler -model_dir biz/model`. The `hz new` drops the `.hz` metadata file at the repo root - commit it, `hz update` depends on it. Then create `script/hz_update.sh` from the template in `backend/idl-workflow.md` - subsequent runs use `./script/hz_update.sh`. Finally run `go mod tidy` and immediately pair it with the mandatory pin `go get github.com/apache/thrift@v0.13.0` plus a second tidy (the pin is mandatory - `backend/backend-skeleton.md` "Known cliffs" explains why; skip it and the step 12 build breaks).
6. **Write the backend support layer, then `container/wire.go`.** Create all hand-written backend support files per `backend/backend-skeleton.md` BEFORE wiring: the DAL (`biz/dal/mysql/init.go` / `probe.go` / `migrate.go` / `page.go`, the hand-written `biz/dal/mysql/model/` structs plus `model/registry.go`, `cmd/generate/generate.go`), `biz/extension/redis`, the feature service (`biz/service/<feature>/service.go`, e.g. `user.NewService`), `biz/mw/`, `biz/consts/errcode.go`, `utils/utils.go`, `localize/`, and `biz/handler/ping.go` + `health.go`. Then run the first `go run ./cmd/generate` to emit the typed `biz/dal/mysql/query/` package - step 12's build depends on it. Only then write `container/wire.go`: its full-profile provider set (`redis.NewRedis`, `user.NewService`, ...) references these files, so wire cannot compile without them. Run `go generate ./container`.
7. **Write `main.go`** following the boot sequence in `backend/backend-skeleton.md` (order chain summarized under "Backend Skeleton Highlights" below).
8. **Write `config/local.example.json`** with the keys main.go reads, plus `test.example.json` / `pre.example.json` / `prod.example.json` siblings with the same shape - `pre` matches the `pre` deploy target in `backend/release-flow.md`, and the example templates are committed. Copy to `config/local.json` for the dev machine; real `*.json` configs stay gitignored.
9. **Scaffold frontend.** First ask the user about SEO requirements to pick the router history mode (see "Routing Mode Decision" below), then choose the design shape and light surface from `frontend/design-profiles.md` (default `micro + soft-white`). Install pnpm via corepack if the machine does not have it: `corepack enable && corepack prepare pnpm@10.0.0 --activate`. Then `cd frontend && pnpm create vite . --template vue-ts` - then remove the Vite scaffold residue, which conflicts with the no-`App.vue` invariant (`frontend/skeleton.md`: the app root is `MainLayout.vue`): `rm src/App.vue src/components/HelloWorld.vue src/style.css`, and empty or delete `src/assets/` (keep it only if a template actually ships assets). Finally apply `frontend/skeleton.md` (axios with refresh interceptor, Pinia auth store, vue-i18n with zh/en, Vue Router 4 with lazy routes), set the profile attributes on both HTML entries, and copy the self-hosted fonts with their licenses.
10. **Wire the embed.** `frontend_embed.go` with `//go:embed all:frontend/dist` per `backend/backend-skeleton.md`. Frontend must build to `frontend/dist/` (and `frontend/dist/admin/` if the admin UI ships in the same repo). If `du -sh frontend/dist/` exceeds ~50 MB, switch to the zstd-archive mode in `backend/embed-strategy.md` instead.
11. **Write `build.sh` + `release.sh` + `Dockerfile` + `Caddyfile` + `docker-compose.yml` + `VERSION` + `script/bootstrap.sh` + `.gitignore`** from `backend/release-flow.md`. The `.gitignore` template (`config/*.json` + `!config/*.example.json`, ...) does double duty: real configs never land in git, and `release.sh`'s rsync `--filter=':- .gitignore'` relies on those same rules to keep them off the server. `build.sh` copies `script/*` into `output/`, and step 12's smoke test runs `./output/bootstrap.sh` - the script must exist here or the smoke test fails. `script/hz_update.sh` (template in `backend/idl-workflow.md`) should already exist from step 5; create it now if you skipped it. VERSION starts at `1.0.0`.
12. **Smoke test.** Full profile only: start the infra first with `docker compose up -d mysql redis` - `NewRedis` / `InitMysql` panic on connect failure at boot, so `/ping` never comes up without them (the lite profile has no such dependency). Then `./build.sh && ./output/bootstrap.sh` should serve `/ping` and the frontend index. Only then commit `chore: initial scaffold`.

If any step fails, fix it before moving to the next. Do not stub.

## Universal Discipline (top-level, applies to every project this skill seeds)

These five blocks are copied verbatim into the new project's `CLAUDE.md` / `AGENTS.md`. They are not Hertz-specific.

The generated `CLAUDE.md` is the ONLY thing that enforces this skill after scaffolding day - later agents will not open the skill. So it MUST ALSO contain, copied verbatim from this file:

1. "The Iron Rule" (IDL first).
2. "IDL-First Workflow (short form)".
3. "Feature Bring-Up Order".
4. Every Red Flag that mentions routes, IDL, `biz/model`, `biz/router`, `biz/dal`, or codegen.
5. The chosen routing mode, infrastructure profile, and design shape/surface decisions.

### Output

- Return code first. Explanation after, only if non-obvious.
- No inline prose. No boilerplate unless explicitly requested.
- Use comments sparingly, only where logic is unclear.
- No em dashes, smart quotes, or decorative Unicode in the code and prose the agent outputs. Plain hyphens and straight quotes only (the UI empty-value placeholder `'—'` rendered by the shared lib helpers is exempt).
- Code output must be copy-paste safe.

### Code

- Simplest working solution. No over-engineering.
- No abstractions for single-use operations. Three similar lines is better than a premature abstraction.
- No speculative features or "you might also want" additions.
- Read the file before modifying it. Never edit blind.
- No docstrings or type annotations on code not being changed.
- No error handling for scenarios that cannot happen.

### Review

- State the bug. Show the fix. Stop.
- No suggestions beyond the scope of the review.
- No compliments before or after.

### Debugging

- Never speculate about a bug without reading the relevant code first.
- State what you found, where, and the fix.
- One pass only. If cause is unclear, say so. Do not guess.

### Execution

- Do not run full builds, e2e suites, or deploys unless explicitly requested. The TDD unit-test loop (`go test` on the packages being changed) is always run - it is part of writing the code, not optional verification.
- When committing, stage only files touched for the task. Always include root `VERSION` when `build.sh`/`release.sh` updated it.

## Stack Lock (non-negotiable defaults for Hertz+Vue projects)

| Layer | Choice | Why this and not the alternative |
|------|--------|----------------------------------|
| HTTP framework | Hertz (cloudwego) | IDL workflow + middleware ecosystem already extracted. Not Gin, not Echo, not chi. |
| API contract | Thrift IDL + `hz` | Generated handlers/router/model give a single source of truth. Not hand-written routes, not OpenAPI-first. |
| DI | Google Wire | Compile-time DI, no reflection. Not fx, not dig, not hand-wired. |
| ORM | GORM + `gorm/gen` | Generated `query` layer for typed queries. Not sqlc, not ent, not raw sqlx. |
| DB | **Full profile:** MySQL 8, utf8mb4, `utf8mb4_unicode_ci`. **Lightweight:** SQLite via `github.com/glebarez/sqlite` (pure Go) | The typed `gorm/gen` query layer stays either way; never `mattn/go-sqlite3` (CGO); PostgreSQL must be re-justified. |
| Cache / lock | `IRedis` seam from `biz/extension/redis`. **Full:** go-redis client. **Lightweight:** in-process `LocalRedis` | Call sites depend only on `IRedis` (`Set/Get/Delete/Exists/Mutex/...`). Do NOT bypass the seam with direct `goredis` calls. |
| Distributed lock | Redsync (REQUIRED dependency in the full profile; the lightweight profile drops it) | Full-profile `NewRedis` builds Redsync unconditionally behind `IRedis.Mutex`/`LockWithKey`; single-host deployments call the same seam, so call sites never change. |
| Auth | JWT in HttpOnly cookies, **access 1h + refresh 30d** | Dual-cookie + `X-Token-Refresh-Needed` response header. Not bearer-in-header, not session cookies. |
| RBAC | **Default (both profiles):** `role` column (`USER` / `MODERATOR` / `ADMIN`) on the user model + `RequireRoleMw`. **Optional escalation:** Casbin with DB-backed policy | Start with static per-route role lists (template: `backend/lite-profile.md`); Casbin only for runtime-editable or per-resource-owner policies (`backend/auth-jwt-casbin.md`). Authenticated `/api/*` accepts USER+; `/papi/*` requires MODERATOR or ADMIN. |
| i18n (backend) | `hertz-contrib/i18n` + `localize/` JSON | `?lang=` query param. Bundles loaded via `localize.GetLocalizeFiles`. |
| Frontend framework | Vue 3 with `<script setup>` + TypeScript | Composition API only. No Options API. |
| Frontend build | Vite 7+ | Not Webpack, not Rspack. |
| Frontend router | Vue Router 4 (lazy routes, **hash mode - `createWebHashHistory`** by default) | One router instance per app entry. Hash mode avoids refresh 404s and removes the need for backend SPA fallback. Switch to `createWebHistory` only when the user confirms a full-site SEO requirement; see "Routing Mode Decision". |
| Frontend state | Pinia | Not Vuex, not custom store. |
| Frontend HTTP | axios with refresh interceptor | Not fetch wrappers, not ofetch (unless replacing the whole interceptor). |
| Frontend reactivity utils | VueUse | Use VueUse before writing custom composables for storage / debounce / fetch / clipboard / breakpoints. |
| Frontend i18n | vue-i18n with `zh.json` + `en.json` | Exactly two locales, identical key sets. Cookie `LOCALE` switches. |
| Frontend visual style | Part 2 of this skill - no component library: native controls + `tokens.css` + `base.css` global styles + shared components | Locked stack, no component library: `base.css` styles bare elements globally; the shared components (BaseDialog, StatusPill, Pagination, CodeEditor/CodeViewer, ...) cover the rest. Element Plus is banned, no exceptions. |
| Package manager | pnpm via corepack | Not npm, not yarn. Lockfile committed. |
| Edge proxy | Caddy 2 | ACME-managed TLS, simple Caddyfile per env. Not Nginx (unless infra mandates). |
| Container orchestration | docker compose (single host) | Not Kubernetes by default. |
| Release | `./release.sh prod` and `./release.sh test` | rsync + remote `docker compose build` + `up -d --force-recreate`. |

## Routing Mode Decision (ask the user before scaffolding the frontend)

Default is **hash mode** (`createWebHashHistory`, URLs like `/#/dashboard`). Everything after `#` never reaches the server, so there are no refresh 404s and no SPA fallback to configure. The cost: the fragment is invisible to search engine crawlers, which only ever see `/`. Ask the user about SEO before writing `routes.ts`:

- **No SEO requirement** (internal tools, login-gated apps): hash mode. Done.
- **Only landing / marketing pages need SEO**: keep the app on hash mode; ship the landing as a separate static page (or prerendered route) outside the Vue SPA. Do NOT switch the whole app to history mode for this.
- **Full-site SEO requirement**: switch to `createWebHistory` AND add an SPA fallback in the Go frontend handler (any GET not under `/api` or `/papi` serves `index.html`). Caddy `try_files` does not apply here - the Go binary serves the embedded files. If crawlers must see rendered content rather than an empty shell, that additionally means prerendering or SSR; scope that as a separate feature, it is not a scaffold default.

Whichever mode is chosen, record it in the new project's CLAUDE.md so later agents do not flip it silently.

Terminology note: the `-[hash]` in built asset filenames (`assets/index-Bx3k9Lm2.js`) is Vite's content fingerprint, on by default and MUST stay on - never strip it via `build.rollupOptions.output` overrides. Hashed assets get long `Cache-Control` (immutable); unhashed `index.html` stays no-cache. This fingerprint is unrelated to the router history mode - do not conflate the two.

## Infrastructure Footprint Decision (ask the user before adding dependencies)

MySQL, Redis, and Casbin are NOT mandatory. Ask the user how the project will be deployed before step 3 of the boot sequence:

- **Full profile** - multi-user production service with external infra available: MySQL 8 + Redis 6, docker compose with mysql/redis services. Casbin stays optional (see the RBAC stack-lock row). Everything else applies as written.
- **Lightweight profile** - single binary, small tool, demo, internal utility, no separate DB/cache services: SQLite via pure-Go `github.com/glebarez/sqlite` (never `mattn/go-sqlite3`, it needs CGO), an in-process `LocalRedis` behind the `IRedis` seam, role-column middleware instead of Casbin. Full templates: `backend/lite-profile.md`.

Two seams exist in BOTH profiles so the cheap one never becomes a rewrite: all cache/lock access goes through `biz/extension/redis.IRedis` (call sites never import `goredis`), and all DB access goes through the generated `dal/mysql/query` layer (only the dialector differs; pick it at scaffold time and stick to it - gen reads the hand-written structs, not a live DB, so generation itself is dialect-agnostic, but after switching dialects you MUST re-run `go run ./cmd/generate` and review the generated SQL for dialect differences). Record the chosen profile in the new project's CLAUDE.md, same as the routing mode decision.

## IDL-First Workflow (short form)

1. Edit `idl/api.thrift` or `idl/papi.thrift` (or a sub-file under `idl/api/`).
2. Run `./script/hz_update.sh`. This calls `hz update` for both api and papi targets.
3. `hz` writes/refreshes:
   - `biz/handler/api/<service>.go` and `biz/handler/papi/<service>.go` - STUBS for new methods, EXISTING bodies preserved.
   - `biz/router/api/*.go` and `biz/router/papi/*.go` - DO NOT edit, will be overwritten.
   - `biz/router/api/middleware.go` - has stubs `_fooMw()` you DO edit to attach per-route middleware.
   - `biz/model/api/*.go` - generated request/response structs.

   Inside a generated handler file you may edit the **function body AND the import block** - hz preserves both on regeneration. You may NOT change the function signature, the `// @router` comment, or move the function to another file.
4. Fill in the function body inside the generated handler. Bind dependencies via `container.Default.<Service>`.
5. **Critical pitfall**: a fresh per-route `_fooMw()` returns `nil`. For any handler that calls `mw.GetJwtId` (or `mw.CheckRoleAny`, available only after the Casbin escalation), ADD `mw.JwtAuthenticationMw()` to that route's middleware. Without it, an expired access token with a valid refresh leaves the context in `JWT_STATUS_REFRESH` and the handler runs unauthenticated.
6. `/papi/*` routes inherit `mw.JwtAuthenticationMw + mw.RequireRoleMw("MODERATOR", "ADMIN")` from `rootMw()` (chain details and the Casbin-escalation swap: "Auth & RBAC" below). You do not re-add them per route.

Full reference (thrift template, naming, paging, error envelope, manual route exceptions): `backend/idl-workflow.md`.

## Codegen Toolchain (install first, never route around)

Version policy: initial scaffold dependencies resolve versionless to compatible releases, then `go.mod` / `go.sum` are committed. Later upgrades are deliberate, one change at a time with generation, build, and tests; no blanket `go get -u`. Codegen tools and the verified cliffs in `backend/backend-skeleton.md` stay pinned because their output is committed source.

```bash
go install github.com/cloudwego/hertz/cmd/hz@v0.9.7      # need v0.9+, preserves edited stubs
go install github.com/cloudwego/thriftgo@v0.4.1          # thrift compiler hz shells out to - pin it, see note below
```

- `hz` does NOT pin thriftgo for you (it silently installs `@latest` when missing). Install thriftgo explicitly BEFORE the first `hz` run; `v0.4.1` is the suggested pin - confirm it on your first `hz update` (clean generation, handlers compile). Bump the trio (hz / thriftgo / `apache/thrift`) deliberately and together: thriftgo's output must match the `apache/thrift` library API.
- Wire runs through `go generate ./container`; the directive in `container/wire.go` uses the module-pinned Wire version rather than a global binary.
- `hz update` is the ONLY way `biz/router/**`, `biz/handler/**` stubs, and `biz/model/**` come into being. `go run ./cmd/generate` is the ONLY way `biz/dal/mysql/query/**` comes into being.
- If the tooling fails (not installed, old version, IDL syntax error), STOP and fix the toolchain or the IDL. Hand-writing any generated surface to "unblock" is never the fallback - that is how every drifted project started.

## Feature Bring-Up Order (every table, every endpoint)

If the thing you need does not exist, the missing step is ALWAYS earlier in this list - never hand-writing a generated surface.

1. **DB schema first, by hand**: write the GORM struct in `biz/dal/mysql/model/<name>.go` (the ONE hand-written model directory). Add it once to `model.MigratedModels()` in `model/registry.go`, the shared registry for GORM gen and boot migration.
2. **`go run ./cmd/generate`** - the typed `biz/dal/mysql/query` layer regenerates. No live DB needed.
3. **Service**: failing test first, then `biz/service/<feature>/service.go` against `dal/mysql/query`, following `backend/data-conventions.md` (typed field methods only, `WithContext`, `FindByPage` + clamp). Services speak DAL models.
4. **IDL**: add req/resp structs to `idl/api/<feature>.thrift`, add the method line to the service block in `idl/api.thrift` or `idl/papi.thrift`.
5. **`./script/hz_update.sh`** - hz appends the handler stub, refreshes routers and `biz/model/**`. Handlers speak thrift models.
6. **Middleware**: attach `mw.JwtAuthenticationMw()` (and friends) in `biz/router/<area>/middleware.go` if the handler reads the authenticated user.
7. **Handler body**: fill the generated stub. `BindAndValidate` -> `container.Default.<Service>` -> map DAL model to thrift model via a helper.

Two model trees (`biz/model/**` hz output vs `biz/dal/mysql/model/**` hand-authored schema) - do not confuse them; see `backend/idl-workflow.md` "Two Model Trees".

## Backend Skeleton Highlights

- `main.go` boot order: `applyBuildRuntimeMode -> LoadConfig -> mysql.InitMysql -> container.InitDefault -> mw.InitMw -> hertz server -> i18n.Localize -> RequestIDMw -> CORS -> Gzip -> ETag -> CSP -> register(h) -> Spin`. Do not reorder.
- Config name comes from `-c <name>` flag, defaulting to `local`. Hot-reload via Viper.
- Local secrets go in `config/local.json` (the file `LoadConfig` reads for `-c local`), which is gitignored; the repo commits only the example templates (e.g. `config/local.example.json`). There is no separate `*.local.json` layer. No real credential lands in git.
- Health endpoints are a fixed set: `/ping` (lightweight alias), `/livez` (liveness), `/healthz` (readiness with dependency probes).
- All `GetConfigStr("...")` keys must exist in every env config or boot fatals. Use `GetConfigStrOptional()` if optional.
- Server max body size: 4GB, `maxServerRequestBodySize` const in main.go (streamed). Make it a config key only if a project actually needs to tune it.
- `OnShutdown`: sleep 2s for in-flight requests.
- Query layer: business code touches ONLY the typed `gorm/gen` DAO (`GetDBQuery().X.WithContext(ctx).Where(q.F.Eq(v))`) - no raw `mysql.DB()`, no string conditions, `FindByPage` + two-sided clamp for lists. Full rules: `backend/data-conventions.md`.
- Cache layer: all cache/lock access goes through the `IRedis` seam; usage has its own rulebook - cache closed facts only (never errors, never open windows like "today" aggregates), singleflight cold reads, fills that survive caller cancellation, canonical keys, one TTL policy per read path. Full rules: `backend/data-conventions.md`.

Full reference: `backend/backend-skeleton.md`.

## Frontend Skeleton Highlights

- **Two entries**: `main.ts` (public, `/`) and `main.admin.ts` (admin, `/admin/*`). Same `frontend/` but separate Vite builds via `VITE_APP_ENTRY=admin`. Both embedded into the Go binary. The admin entry reuses the same `MainLayout` template (never a forked copy - the "never fork a template" rule applies); only `routes.admin.ts` and the nav config differ.
- **Auth flow**: `stores/auth.ts` calls `api.getUserInfo()` once at startup. On **403 + `X-Token-Refresh-Needed: true`** the axios interceptor calls `/api/user/refresh` exactly once, then replays the original request. **Do NOT auto-refresh on the very first `getUserInfo` failure** - guests get plain 401 (no header) and refreshing for them spams 401 logs.
- **Guarded fetches**: any component that reads protected `/api/*` data must wait for `authStore.loading === false && authStore.user !== null` before triggering the request.
- **Lazy routes**: every page is `() => import(...)`. No eager imports for sub-pages.
- **i18n**: `vue-i18n` with `zh` and `en` only. Cookie `LOCALE` controls. Keys live under `common.*`, `<feature>.*`, `admin.*`. zh and en MUST have identical key sets.
- **UI stack: Part 2 of this skill, no component library.** `tokens.css` + `base.css` (verbatim from this skill, imported in that order) are mandatory; they style bare `button` / `input` / `select` / `table` globally and ship every shared class. Controls are native elements plus the shared components (`BaseDialog`, `StatusPill`, `Pagination`, `CodeEditor`/`CodeViewer`, ...).

Full reference: `frontend/skeleton.md`.

## Auth & RBAC

JWT cookie auth applies to BOTH profiles. The default RBAC is the simple one - `role` column + `RequireRoleMw` (`backend/lite-profile.md`). Casbin is the OPTIONAL escalation for runtime-editable policies (`backend/auth-jwt-casbin.md`).

- Cookies: `ACCESS_TOKEN` (HttpOnly, SameSite=Lax, **TTL 1 hour**), `REFRESH_TOKEN` (same, **TTL 30 days**). `Secure` follows the `jwt.cookie_secure` config (set true whenever the deployment is reached over TLS, including behind a TLS-terminating proxy; off for plain-HTTP local dev). `server.tls` only controls the app's own TLS listener. Names configurable in `biz/consts`.
- Middleware chain on `/papi/*`: `JwtParseMw -> RateLimitMw -> JwtAuthenticationMw -> RequireRoleMw("MODERATOR", "ADMIN")` (attached in `biz/router/papi` `rootMw()`; the `ModeratorOrAdminMw` sugar in `biz/mw/role.go` is equivalent). The Casbin enforcer middleware replaces the role check only after the optional Casbin escalation.
- `/api/*` root middleware: `JwtParseMw -> RateLimitMw`. Per-route adds `JwtAuthenticationMw` if the handler needs an authenticated user.
- Casbin policy is **DB-backed** via `gorm-adapter` against the same MySQL instance. Three subjects: `USER`, `MODERATOR`, `ADMIN`. Policy mutations publish a `casbin_notify` event on Redis; every replica subscribes and reloads on receipt - do not poll.
- JWT trims cookie whitespace; reads `type` claim as a string; if access is unusable but refresh validates, returns `JWT_STATUS_REFRESH` (HTTP 403 + `X-Token-Refresh-Needed: true`) - never `JWT_STATUS_FAIL`.
- `isJwtUserActive` denies auth on any DB error including `ErrRecordNotFound`. A flaky DB must not grant access.
- Authorization does not stop at route middleware: one shared service-layer validator binds every entry point (public API, admin API, agent/tool APIs, internal jobs), destructive ops are state-scoped and audit-preserving, scoped credentials filter lists by scope, and a route-to-policy coverage test asserts both directions. See "Authorization Rules Beyond Middleware" in `backend/auth-jwt-casbin.md`.

Full reference: `backend/auth-jwt-casbin.md`.

## CI/CD and Release

- `build.sh` is local-incremental: hashes frontend inputs to skip frontend rebuilds and Go inputs to skip Go rebuilds (see `backend/release-flow.md` for the full hash input list). Bumps `VERSION` patch when sources changed unless `APP_SKIP_VERSION_BUMP=1` or a tag-driven `GITHUB_REF_NAME=vX.Y.Z` is present.
- `release.sh prod` reads `VERSION`, bumps patch, writes VERSION, rsyncs to the server, runs `docker compose build` then `docker compose up -d --force-recreate <service>` and `caddy`. `release.sh test` / `pre` do the same against `app-test` / `app-pre` WITHOUT writing VERSION locally.
- Dockerfile is three stages: `node:24-slim` (pnpm frontend build) -> `golang:1.26-bookworm` (go build with `-ldflags "-s -w -X main.version=..."`) -> `debian:bookworm-slim` runtime with `tzdata` and `ca-certificates`.
- `docker-compose.yml` exposes mysql/redis on `127.0.0.1` only. Caddy is the only edge proxy on 80/443. ACME state lives under `./data/caddy/` - back up that volume.

Full reference: `backend/release-flow.md`.

## Testing

**Required cadence**: TDD for any feature or bugfix. Write the failing test first; watch it fail; minimal implementation; pass; refactor.

**Required coverage** (the floor):
- Every `biz/service/<feature>` package: table-driven `go test` for the public methods, run against a disposable DB via `testutil.MustDB` + the `NewWithQuery` test seam (there are no gen mocks). Handler-level tests use `httptest`; multi-write transactions prove rollback on a later failure.
- Every reusable backend middleware: unit test the happy + refusal paths.
- Model/migration contracts: every registered model has a generated DAO; migrations cover fresh/old schema, idempotent rerun, and invalid-definition or dirty-data refusal. Soft-delete models cover hidden deleted rows, unique-name recreation, and tombstone retention.
- Shared frontend async and interaction behavior: request cancellation/stale response/timeout, URL-state ownership and deep links, `SplitResizer`, `TableEmptyRow`, and `Pagination` normalization. Page snapshots remain optional.

**Allowed shortcuts**: frontend page-level snapshot tests and E2E are optional (Playwright if added). They do not replace the shared behavior tests above.

**Test invariants**:
- Unit tests do not hit third-party services. Stub them via the `biz/extension/<ext>` interface.
- Dialect- or migration-sensitive integration tests use isolated real databases, never production data. MySQL uses `TEST_MYSQL_DSN` and skips when unset, so `go test ./...` works on a clean clone. Lightweight-profile tests use temp-file SQLite (`t.TempDir()`), the production dialect there.
- One backend command MUST work: `go test ./...`. CI runs it plus `go vet ./...`, gofmt, and generated-source drift checks. The frontend full gate is `pnpm check`.

Full reference: `backend/testing-conventions.md`.

## Explicitly Excluded from Scaffold

These are NOT part of the scaffold. If the user asks for any of them, treat as a new feature and require explicit scope, not a scaffold default.

| Excluded | Why |
|---------|-----|
| Desktop/offline variants (Wails, `offline_*` files, `//go:build offline`) and source obfuscation (`garble`) | Out of scope for this scaffold. |
| MCP server endpoints | Domain-specific feature - add only when the product actually exposes tools to agents. |
| Embedcompress (`cmd/embedcompress`, `frontend/dist.tar.zst`) - OPTIONAL, OFF by default | Switch to the zstd-archive mode only when `frontend/dist/` exceeds ~50 MB unpacked or bandwidth is constrained. Trigger criteria and implementation: `backend/embed-strategy.md`. |

---

# Part 2 - Style (the visual system)

A Vue 3 design system with no component library: native HTML controls styled once by a global `base.css`, a full token file, a fixed app shell, split list+detail layouts, and a set of hand-rolled shared components and composables. Flat paper surfaces with a single cyan accent (Catppuccin swatches remain for charts/graphs), spring easing, Chinese ops-console copy.

## The Iron Rule (Part 2)

**Do not invent tokens. Do not install a component library. Do not build page-local variants of things the system already has.**

This skill ships a complete token file, a complete global stylesheet, layout templates, and component/composable contracts. You copy those, you do not redesign them. Cross-page consistency is the top priority: before adding a page, component, or interaction, find how sibling pages already do it and copy that.

If you find yourself writing `#3b6cff`, `border-radius: 8px` on a button, styling a `<select>` in a page's `<style scoped>`, or hand-rolling a dialog/tabs/pagination, **STOP - you are reverting to default-agent behavior**. Read the relevant reference file and use the existing pattern.

## Mandatory Workflow

Follow these steps in order for any task this part applies to. Structural code is copied, never written: the template set is the contract, and hand-typed shells/dialogs/icons are how alignment bugs are born.

1. **Choose the design profile only at project or redesign scope**: offer the shape and surface options in `frontend/design-profiles.md`. Default to `micro + soft-white` when the user has no preference. Record the decision in the project's `AGENTS.md` / `CLAUDE.md` and set the same root attributes on every HTML entry. Do not ask again for each page. An existing project keeps its current palette unless the user explicitly requests a redesign.
2. **Copy the stylesheets and fonts**: `frontend/tokens.css` + `frontend/base.css` to the project's `src/styles/` (plus `frontend/code-editor.css` if the project has any code/JSON input or viewer). Copy `frontend/assets/fonts/` to `public/fonts/`, including licenses, and preload both fonts from each HTML entry. Styles ship as a unit. All colors/radii/shadows/fonts/spacing/durations MUST come from these variables. Never restyle bare elements per page.
3. **Copy `frontend/templates/` as a unit** into the project (keep the flat set together, e.g. under `src/base/` - its imports are sibling-relative). Customize ONLY the `EDIT:` markers in `MainLayout.vue` (nav arrays, brand name, storage key prefixes). Verbatim copy is what makes alignment a construction-time guarantee instead of a verification-time hope: ANY local modification beyond the `EDIT:` markers voids that guarantee. If a template feels wrong, fix it upstream in this skill and re-copy - never patch it inside the project. If a template does not cover a need, add a sibling file per the `frontend/components/` docs; never fork a template.
4. **Verify the shell** against `frontend/layouts/app-shell.md`: fixed frame, flat opaque surfaces, icon sidebar with desktop collapse persistence, floating nav panel under 960px, shell-rendered page header, dark toggle, errorBus toast stack. The sidebar collapse and nav panel specs in that file are normative.
5. **For each page**, pick the page pattern from "Page Patterns" below: list+edit pages use the split layout, long pages scroll inside `.page-stage`, create forms and one-off results use `BaseDialog`. Pages never render their own `h1`.
6. **For each control type**, use the global style or shared component: buttons/forms/selects/chips/loading/empty-state from `frontend/components/*.md`, dialogs from `frontend/components/dialogs.md`, code/JSON from `frontend/components/code.md`.
7. **Before declaring done**, scan the page against the "Red Flags" list below on the home page and one list page, in both themes and the selected light surface profile. Check contrast, sticky columns, control alignment, and mobile stacking.

## Shared Constants and Profile Choices

The accent, typography roles, control geometry, and interaction constants below are fixed. Shape and light surfaces are project-level choices from `frontend/design-profiles.md`; pages never override either.

| Constant or choice | Value | Why |
|---------|-------|-----|
| Accent cyan | `#00a8cc` (light) / `#00d9ff` (dark) | `--accent`; NOT the old `#1e66f5` blue, NOT `#3b6cff` |
| Accent strong | `#008fb3` (light) / `#00b8d9` (dark) | `--accent-strong`; button hover background and light-theme text links |
| Accent ring/glow | `rgba(accent, 0.15)` / `0.25` | `--accent-ring` (focus ring), `--accent-glow` |
| Text primary | `#1e1e1c` (light) / `#e8e8e6` (dark) | neutral text, NOT `#4c4f69`, NOT `#0f172a` |
| Text muted | `#666661` (light) / `#94948f` (dark) | readable secondary information; verify contrast on every surface profile |
| Light surfaces | `data-surface="white"`, `"soft-white"` (default), or `"warm-paper"` | flat, opaque, selected once per project; dark surfaces remain the shared dark palette |
| Shape | `data-shape="micro"` (default), `"rounded"`, or `"square"` | one radius scale per project; roles stay fixed across shapes |
| Borders | `rgba(30, 30, 28, 0.08)` soft / `0.16` strong (dark: white 0.08/0.16) | 1px everywhere; the ONLY card chrome |
| Display font | `--font-display`: `'Plus Jakarta Sans'` first, then PingFang SC / Noto Sans SC / Microsoft YaHei | self-hosted WOFF2, no runtime font CDN |
| Mono font | `--font-mono`: `'Geist Mono'` first | data, code identifiers, and numeric content; not ordinary Chinese labels |
| Type scale | `--text-xs/sm/md/base/lg/xl/2xl` = `11/12/13/14/16/20/28px` | shared CSS and page styles use the scale; no off-scale values such as 12.5px |
| Button geometry | `--radius-sm`, `min-height: 36px`, `padding: 0 14px`, solid `var(--accent)` | same baseline height as inputs/selects; no gradients or page-local shapes |
| Input geometry | `--radius-sm`, `min-height: 36px` (textarea: `--radius-md`, min 80px); date/time/datetime-local get `line-height: 32px` | global style in base.css |
| Card/shell radius | `--radius-md` for `.card`, dialogs, sidebar, header | profile-driven; compact 14px 16px card padding |
| Easing | `--ease-spring`: `cubic-bezier(0.32, 0.72, 0, 1)` | NOT `ease`, `ease-in-out`, or linear - EXCEPT infinite looping animations (e.g. the spinner rotation), which may use `linear` |
| Durations | `--duration-fast` 0.18s / `--duration-base` 0.24s / `--duration-slow` 0.36s | no raw `0.2s`; honor `prefers-reduced-motion` |
| Sidebar width | `232px` expanded, `72px` collapsed (desktop only; <=960px uses the floating nav panel) | fixed grid, persisted in localStorage |
| Focus ring | `0 0 0 3px var(--accent-ring)` (`--focus-ring`) | on every input/select/textarea focus |
| Shadows | `--shadow-soft/floating/dialog/tooltip` + `--shadow-sticky-col` | cards have NO default shadow; hover accents, floating layers, and the sticky action column only |
| Z-index | `--z-sticky-col/head/pagination/actions-head` 1-4, `--z-shell-header` 10, `--z-loading` 50, `--z-popover` 300, `--z-nav-panel` 8900, `--z-dialog` 9000, `--z-confirm` 9001, `--z-menu` 9500, `--z-toast` 9600 | token ladder; no raw numbers |

`tokens.css` carries the full token set, including the `--ctp-*` Catppuccin palette, `--graph-*` graph colors, and `--space-*` spacing scale. `frontend/design-profiles.md` is normative for the exact profile maps and upgrade rules.

## Required Signature Elements

Every app in this style MUST have:

1. **Flat paper surfaces** - the paper base palette (`--app-bg` + opaque `--surface-*`) with a single cyan accent. No ambient gradient, no grain/noise overlay, no decorative gradients anywhere.
2. **1px-bordered cards with no default shadow** - `.card`, sidebar, and header are `--radius-md` surfaces whose only chrome is `1px solid var(--border-soft)`. Shadows exist ONLY in three places: as hover accents (primary button hover `--accent-shadow`, danger button hover `--danger-shadow`), on floating layers (popover, nav panel, dialog, confirm, menu, tooltip, loading overlay), and as the sticky action column's structural shadow (`--shadow-sticky-col` on `td.row-actions` / `th.col-actions`). `backdrop-filter` does not exist in this system.
3. **Z-index from the token ladder** - `--z-sticky-*` (1-4), `--z-shell-header`, `--z-loading`, `--z-popover`, `--z-nav-panel`, `--z-dialog`, `--z-confirm`, `--z-menu`, `--z-toast`. No raw z-index numbers in components or pages.
4. **Fixed app shell** - by default `.shell-layout` pins to `100vh` and never scrolls; scrolling happens only inside regions (`.page-stage` for normal pages, `.split-master` / `.split-detail` on split pages). Never make the window or `.shell-main` scroll. The sole exception is the <=760px document-scroll unlock built into the `MainLayout.vue` template (see "Split layout" below). Under 960px the nav is a floating panel (see `frontend/layouts/app-shell.md`), never a drawer.
5. **Shell-rendered page header** - the shell header renders `page-eyebrow` + `page-title` + `page-sub` from the nav definition / route meta. Pages never render their own `h1` or page header. (Card-level `.section-head` with a 12px muted description line is a legal pattern and not a violation.)
6. **Icons from `icons.ts` only** - every nav item carries an `IconName`; every close button is `<Icon name="x">`; no emoji, and no bare `x` / `+` / dot characters standing in as icons. (Text arrows inside button copy - like Pagination's `‹ 上一页` / `下一页 ›` - are labels, not icons, and are allowed.)
7. **CSS variables only** - no hardcoded hex/rgba in components or pages; everything goes through `var(--...)` from `tokens.css`.
8. **Both light and dark mode** - `:root.dark` in `tokens.css` carries a full override set; toggle via `document.documentElement.classList.toggle('dark')` + `localStorage['theme']`. Every page is checked in dark mode before done.

## Page Patterns

### Split layout (list + edit pages)

- `.split-layout` > `.split-master` (list, `flex: 0 0 clamp(520px, 40vw, 720px)` via `--split-master-width`, scrolls internally, pins its `Pagination` at the bottom) + `.split-detail` (edit pane, fills the rest, scrolls internally).
- In the 761-960px band the split STACKS vertically (`.split-layout` becomes a column, `.split-master` takes `flex: 0 1 40%`, `.split-detail` the rest); at 760px and below it also stacks and the whole document scrolls. base.css ships the stacking fallbacks; the <=760px document-scroll unlock (`.shell-layout` / `.shell-main` height auto, `.page-stage` overflow visible) lives in the `MainLayout.vue` template `<style>` block. Both ship with the skill - do not write per-page versions.
- `.split-detail` direct children must stay `flex-shrink: 0`; a single primary long block (JSON / raw payload / large text) gets `.grow-fill` to fill remaining pane height - never when multiple long blocks coexist.
- Unselected state shows `.split-detail-empty` (dashed border placeholder).
- Row selection is whole-row click, never a dedicated select column: the row carries `:class="{ selected: ... }"` + the base.css `.row-link` class (ships `cursor: pointer` and the focus outline - no inline `style="cursor:pointer"`) + `tabindex="0"` + `role="button"` + `@click` and `@keydown.enter/space.prevent` handlers; action buttons inside the row use `@click.stop`. Note the visible `tr.selected` highlight (accent background + marker shadow) is scoped to `.split-master` in base.css. Selection persists in the URL via `useRouteBackedSelection`.
- Detail pane header uses `.detail-head` (title left, actions right in a `.detail-actions` flex row); op feedback (`.op-ok` / `.op-err`) sits below the form action row.
- When a page accumulates multiple stacked cards/splits, split it into tabs (see below) or separate pages. One page carries one functional area.

### Toolbar spec (every list page, in this order)

1. `+ 新建 X` primary button at far left (management pages).
2. Filters, each with a sibling `label`; `select` / inputs stay `width: auto` (base.css `.list-toolbar` rule - do not let the global form `width: 100%` leak in). Only a primary search box may span remaining width via `input.toolbar-search`.
3. `刷新` muted button with `:disabled="loading"`.
4. `<Loading inline>` (never the fullscreen overlay on list refresh).
5. `.op-err` span for the fetch error.
6. `共 N 条` as `.muted-text` with `margin-left: auto` - ONLY on lists without a `Pagination`. When `Pagination` is present it already renders `from - to / 共 N 条`, so the toolbar omits the counter.

Text filters that trigger fetches debounce ~250ms and clear their timer on unmount. Clearing a search input refetches the unfiltered list - never leave stale filtered options on screen.

### Table spec

- Cells default to `white-space: nowrap`, columns size to content; `table` has `min-width: 720px` and `overflow: clip`; the scroll container (`.split-master` / `.card` / `.table-scroll` - wrap bare tables in `<div class="table-scroll">`) scrolls horizontally. Never `table-layout: fixed` or per-cell ellipsis to squeeze columns.
- `th` is sticky to the top of its scroll container, 11px muted `--font-display` with normal case and letter-spacing, and an opaque `--surface-sticky-header` background. Chinese labels never use uppercase transforms.
- Action column: header `<th class="col-actions">操作</th>`, cells `td.row-actions`, both sticky right with opaque background.
- Long free-text columns get `.cell-truncate` (360px ellipsis); columns that must wrap get `.cell-wrap`. Time/count/duration/token columns get `.cell-num` on both `th` and `td` (right-aligned `tabular-nums`).
- Row actions are `muted btn-sm` inside `.row-actions`: at most 2 inline; with 3+ keep the single most-used non-danger action inline and move the rest into `RowActionsMenu`; danger actions always go into the menu and are always paired with `useConfirm`.
- Enable/disable is always `.toggle-btn` (lists and forms, never checkboxes). Condition-disabled buttons carry a `title` with the reason.
- `TableEmptyRow` renders an error row and emits `retry` on failure; only a successful `count === 0` response renders `EmptyState`. Standalone empty areas use `EmptyState`.

### Form spec

- Field labels use `.field-label`; required fields add `<span class="required-mark">*</span>` and validation shown next to the field.
- Validation timing follows `useFormValidation` (`submitted` + `blur`/`show`/`reset`): dialogs and pages open clean with no errors shown; an error appears only after the first submit attempt or that field's blur, and clears as soon as it is fixed.
- JSON inputs use `JsonField` (validates on input, surfaces `JSON 格式错误:...` below the field).
- Submit buttons are disabled while the request is in flight, never pre-disabled by validation state.
- Date/time inputs are always the native `input[type='datetime-local']` (or `date`/`time`) styled by the shared base.css block — never page-local date pickers, calendar popups, per-page restyle/resize, or composite hour/minute fields (full rules: `frontend/components/forms.md` "Date and time inputs"). Relative ranges (最近 1 小时 / 24 小时 / 7 天 / 30 天) use a `<select>` preset; `input[type='month']` is the sanctioned monthly picker (`style="width:auto"`); fields bind with `v-model` and convert via `parseInputTs` on submit.
- `.edit-grid` (label column default 120px) for aligned edit forms; `.check-inline` for inline checkbox rows.

### Page style discipline

- A page's `<style scoped>` never redefines or specializes a shared base.css class (no local `.muted-text` / `.pill.risk-low` / `.detail-head` / `.split-detail` variants). Allowed extensions: style a page-local child under a shared class (`.table-scroll table`, `.split-detail > .form-scroll`), or add a standalone modifier class in markup next to the shared one.
- Inline `style` must not restate a value the class already sets (`font-size:12px` on `.muted-text` is dead code). The spec-sanctioned inline exceptions: `style="width:auto"` on `input[type='month']`, and `margin-left:auto` on the `共 N 条` counter.
- Vue component attributes satisfy `vue/max-attributes-per-line` and `vue/first-attribute-linebreak`.

### Tabs spec

- `.tabs` from base.css: flex row with `gap: 4px`, a 1px `--border-soft` bottom border, and `margin-bottom: var(--space-3)` (removed when it is a direct card child).
- `.tabs button` overrides the global button style: `padding: 6px 12px`, `margin-bottom: -1px`, `font-size: var(--text-base)`, `font-weight: 500`, `line-height: 22px`, no border except a 2px transparent bottom border, top corners rounded `var(--radius-xs)`, transparent background, `--text-secondary` text.
- Hover: `--accent-soft` background + `--accent-strong` text.
- Active in light themes: primary text + `font-weight: 600` + `--accent-soft` background + 2px accent bottom border. Dark-theme hover/active text uses `--accent`.
- ARIA contract is mandatory: container `role="tablist"`, each button `role="tab"` + `aria-selected` + `aria-controls` + `tabindex` (0 on active, -1 otherwise), each panel `role="tabpanel"` + `aria-labelledby`.
- `useTabs` provides only keyboard navigation (ArrowLeft/ArrowRight wrap, Home/End); the page owns selection state.
- Tab state persists in the route query.

## Alignment Contract

**The one root rule: zero magic numbers, zero local alignment patches.** Never hand-tune `line-height` / `padding` / `margin` / `width` on an individual control to make it align. When a control looks misaligned, the correct action is to fix the shared layer (`base.css`, or the template upstream in this skill) so every control of that kind benefits; a page-level patch is how the next ten misalignments are born. Any "this component is special" local style is a bug seed. For the same reason, verification is component-agnostic: it scans invariants (the "Red Flags" list below) over every control on the page instead of asserting what any one component should look like.

**What the shared layer already guarantees** - copy it verbatim and you inherit all of this; each rule also names the typical way hand-written pages break it:

**a. Control geometry.** Buttons, text inputs, and selects share `min-height: 36px`; buttons and inputs use `--radius-sm`, while textareas use `--radius-md`. The project shape profile controls the actual radii. Single-line inputs vertically center text through their content box (`padding: 0 14px` + `min-height`), no `line-height` tricks - EXCEPT the `date` / `time` / `datetime-local` family, which base.css gives an explicit `line-height: 32px`: under custom fonts these inputs otherwise pin their text to the top of the box. Do not remove that rule when copying. Typical breakage: a one-off input with `height: 32px` or a bare `line-height: 1` sitting next to spec controls. Compact actions use the shared `btn-sm` variant instead of locally cancelling button height.

**b. Icons.** Every icon has a fixed box: nav 18x18, nav subitems 15x15, in-button and dialog icons 14-16px, always with `flex-shrink: 0`. No layout change may scale an icon, and no flex row may compress one. The collapsed sidebar removes labels from the DOM (`v-if`) instead of width-animating them. Typical breakage: an icon inside a flex row without `flex-shrink: 0` gets squeezed as space tightens.

**c. Toolbar label + control groups.** A filter is a sibling `label` plus its control on one line: `display: inline-flex; align-items: center` (base.css `.row > label` / `.list-toolbar > label:has(> input)`). Toolbar inputs/selects stay `width: auto`; only `input.toolbar-search` grows. Typical breakage: the global form `width: 100%` leaks into the toolbar so every filter becomes its own row, or flex stretches an input off its label's baseline.

**d. Numeric text.** Numbers, dates, IDs, durations, latency, percentages: `.tabular` (`font-variant-numeric: tabular-nums`), or column widths jitter on every refresh. `.pill` already carries it; table cells and metric values need the class explicitly (`.tabular` lives in `tokens.css`).

**e. Sidebar collapse.** The full spec lives in `frontend/layouts/app-shell.md` and is normative: grid-track transition 232px to 72px (desktop only), labels leave the DOM, fixed-box icons centered with unchanged item height, group headers hidden and subitems flattened, `title` tooltips on the rail. Typical breakage of a hand-rolled rail: width-animated labels, unsized icons, visible group headers, width transitions.

**f. Shadows.** Flat surfaces; shadows only as hover accents, on floating layers, and on the sticky action column - the full three-category enumeration is Signature Elements #2. Typical breakage: a card with a default shadow, or a `.muted` hover shadow.

**g. Stat blocks.** Dashboard metrics use the base.css pair `.stat-value` (28px, weight 700, `line-height: 1`, `letter-spacing: 0`, text-primary) + `.stat-sub` (muted 12px), with `.tabular` on the value when it is numeric. Typical breakage: one-off font sizes (30px/32px) or a bold colored value that fights the page hierarchy.

## Shared Components and Composables

Use, never rebuild. Two tiers:

- **Template tier** - files under `frontend/templates/`, copied as a unit into ONE directory (e.g. `src/base/` - imports are sibling-relative, see "Mandatory Workflow"). The headings below are role groupings, not physical paths.
- **Contract-only tier** - no template ships; the skill documents the contract (in `frontend/components/*.md` and the entries below) and the project implements each one once, when first needed. Names are fixed; the implementation is per-project.

Components:
- TEMPLATE: `Icon.vue` + `icons.ts` (the only icon source), `Loading`, `Pagination` (default `sizes: [20, 50]`, aligned with the backend `MaxLimit = 50` clamp in `backend/data-conventions.md`; stale sizes normalize to an allowed value), `EmptyState` + `TableEmptyRow` (error detail + retry, empty only after success), `BaseDialog` + `ConfirmDialog`, `SplitResizer` (pointer, keyboard, reset, persisted ratio). Behavior details: `frontend/components/*.md`.
- CONTRACT-ONLY: `StatusPill` / `SeverityPill` (`size="sm"`) / `TierPill` (labels and classes only from `lib/labels.ts`), `RowActionsMenu` (overflow menu, `dots` trigger), `MarkdownText` (markdown-it, links forced `target="_blank" rel="noopener noreferrer"`), `HoverPopover` (rich hover; native `title` only for one-liners), `JsonViewer` (field-list default, raw toggle persisted; large payloads default collapsed via `collapsedSummary`), `JsonField` (JSON input, live parse validation), `CodeEditor` (editable: textarea over highlighted `pre`, `lib/codeHighlight.ts` tokenizer), `CodeViewer` (read-only CodeMirror 6; never Monaco). Contracts: `frontend/components/code.md` / `chips.md` / `dialogs.md`.

Composables:
- TEMPLATE: `useFetch` + `usePaginatedFetch` (new refreshes abort old requests, Axios `AbortSignal` propagation, `cancel()`, optional `timeoutMs`, scope-disposal cleanup, and a `seq` guard so stale responses never overwrite newer ones), `useTabs` (keyboard nav for `.tabs`), `useConfirm`, `useFormValidation`, `errorBus` (`'ok'` auto-dismisses 4s, `'err'` 8s with manual close; identical re-show refreshes the timer; success feedback must pass `'ok'` explicitly), `dialogStack` (Esc closes only the top).
- CONTRACT-ONLY: `useRbac` (`canWrite` / `canAdmin` gate buttons), plus the two URL-state composables:
  - `useRouteQuerySync<T extends Record<string, string>>(defaults: T): { query: T; setQuery(patch: Partial<T>): void; resetQuery(): void }`
  - `query` is initialized from the hash query merged over `defaults`; `setQuery`/`resetQuery` write back via `router.replace` (never `push` - filter changes must not pollute history); values equal to their default (or empty string) are omitted from the URL. Filter changes reset page to 1.
  - Each composable declares the keys it owns and preserves sibling/parent keys by default. External back/forward navigation rehydrates state; async `onChange` work is cancellable or sequence-guarded and disposed with the scope. Numeric parsing supports a configurable minimum rather than assuming every number must be positive.
  - `useRouteBackedSelection(param = 'id'): { selectedId: ComputedRef<string>; select(id: string): void; clear(): void }`
  - `selectedId` reads the hash query param; `select`/`clear` update it via `router.replace` while preserving other query keys. Master-row `:class="{ selected }"` and the detail pane both bind to it, so a deep link opens with the row preselected. Deep-link loads discard stale responses and handle a missing row explicitly.

Lib:
- TEMPLATE: `lib/time` (`time.ts` - `fmtTs` / `fmtIso` / `fmtRel` / `fmtAge` / `fmtDur` / `fmtInputTs` / `parseInputTs`; empty/invalid input renders `'—'` / `''` / `0` per helper), `lib/clipboard` (`copyText`), `lib/errors` (`errorMessage`).
- CONTRACT-ONLY: `lib/labels.ts` (the ONLY registration point for status/severity/risk/tier labels and pill classes), `lib/codeHighlight.ts` (regex tokenizer behind `CodeEditor` / `JsonField`).

Data and API conventions:
- int64 ids: response integers that may exceed 2^53 MUST be stringified at the API layer (backend DTO or client-side transform - implementation is per-project). Use ids as strings in paths and comparisons; keep ids as numbers in request bodies.
- RBAC: roles rank USER < MODERATOR < ADMIN (the same names as the backend `role` column and `RequireRoleMw`); gate write/admin controls with `v-if="canWrite"` / `v-if="canAdmin"` from `useRbac` AND carry `meta.requiresRole` on the route as the second gate. Both layers.
- Route-level code splitting: pages lazy-load (every page is `() => import(...)`; no eager imports for sub-pages).

## Copy and Writing Style

- UI copy is Chinese. API field names and product terms (cron / Schema / Webhook / Runner, etc.) stay English. When the project has vue-i18n (Part 1 frontend skeleton), the Chinese copy of page/business code lives in `zh.json` keys, not template literals. The locked shared templates (`MainLayout` / `Pagination` / `Loading` / `BaseDialog` / `useConfirm`) are the exception: they ship their Chinese ops copy by design and stay verbatim - do not refactor them onto `t()`.
- Every status/severity label is registered once in `lib/labels.ts`; pages never invent their own label maps.
- Empty values render `'—'` (via `lib/time` / `statusLabel` and friends), never blank cells or `null`.
- API sentinel numbers never render raw: `-1` meaning unlimited renders `无限`; an unknown or unavailable count renders `未知` / `总数未知`.
- All time display goes through `lib/time` helpers.
- User-facing errors read `'动作名失败:' + errorMessage(e)` (note the halfwidth colon), e.g. `保存失败:...`.
- Prose rules for all user-facing text (docs, messages, labels):
  - Use standard terms; one thing has one name everywhere. Common engineering terms (kill / heartbeat / fallback / throttle) stay as-is.
  - No metaphors, no personification of system components, no marketing tone.
  - Neutral category labels (问题 / 现象 / 影响 / 结果); state facts directly.
  - No "是 X 不是 Y" contrast phrasing.
  - No colloquialisms (坑 / 刷 / 白跑 / 卡死 / 搞定).
  - An unexplained failure reads "原因未查明"; never fill in an unverified guess.
  - LLM usage is `token`; credentials are `令牌`.

## State Persistence

- List page state (filters, pagination, selection, tab) lives in the URL hash query via `useRouteQuerySync` / `useRouteBackedSelection`. A shared link must reproduce the view.
- localStorage keys used by the system (pick a per-app prefix, shown as `<app>` below):
  - `theme` - `dark` / `light`.
  - `<app>.sidebar.collapsed` - sidebar collapse flag.
  - `<app>.nav.expanded` - JSON array of expanded nav group keys.
  - `<app>.jsonviewer.mode` - `list` / `raw` for `JsonViewer`.
  - `<app>.split.width.<key>` - `SplitResizer` ratio for one stable feature key.
- sessionStorage: `<app>.token` - ONLY in deployments that authenticate with an `Authorization: Bearer` header instead of the cookie scheme (e.g. behind an authenticating upstream). The default HttpOnly-cookie auth stores nothing in web storage; a bearer token, when one exists, is sessionStorage only, never localStorage.

---

## Rationalizations to Reject (both parts)

| You might think | Reality |
|----------------|---------|
| "IDL is overkill for one endpoint, I'll just add a route in `router.go`" | The next agent will. And the one after that. Bypass the contract once and the contract is dead. Add it to the IDL. |
| "`hz update` failed / hz isn't installed, I'll register this route manually just this once" | Fix the toolchain or the IDL. Hand-registering "just this once" is exactly how projects end up with anchor IDLs, `manual.go` route files, and handlers that ignore hz models. |
| "I'll skip Wire for now and `var Default = &Container{ ... }` manually" | Wire takes 5 minutes to set up. Hand-wiring will be ripped out the moment a service needs another service. |
| "GORM gen is too much ceremony, I'll just hand-write queries" | The point of the typed `query` layer is that `where("active = ?", true)` typos are compile errors. Without it, schema drift goes undetected. |
| "PostgreSQL is more modern, I'll switch" | Switching the DB means re-justifying `gorm/gen`, the migration path, MySQL-specific config keys. If you have a reason, raise it as a decision. Don't default-switch. |
| "Let me add Tailwind / Ant Design Vue / Element Plus, it's faster" | The UI stack is locked: Part 2 tokens + `base.css` global styles + shared components, no component library, no exceptions. Tailwind/AntD/Naive/Element Plus bring conflicting resets and design tokens, and reaching for a library is exactly how per-page bespoke buttons and divergent selects ship. If the user wants a different library, they will say so. |
| "Native select/input needs a component library to look right" | `base.css` IS the component library. Bare elements are styled globally by design; a page-local restyle is the bug. |
| "This page is special, it needs a local style" | Cross-page consistency outranks page preference. Find the shared pattern; if none exists, extend the shared layer, not the page. |
| "I'll install Monaco / CodeMirror for editing" | `CodeEditor` (editable) and `CodeViewer` (read-only CodeMirror 6) already cover both directions. |
| "I'll hand-roll a dropdown/tooltip/popover" | `RowActionsMenu`, `HoverPopover`, and `dialogStack` already define the interaction contract. |
| "I'll do JWT in `Authorization: Bearer ...` header, simpler" | HttpOnly cookies + refresh is the documented flow. Bearer header on a browser app means XSS = token theft. |
| "I'll add `auto-refresh on app load` for nicer UX" | Don't. Auto-refresh on load also fires for guests, causing 401 spam. The auth store waits for `getUserInfo`; refresh only on the X-Token-Refresh-Needed signal. |
| "I'll write the handler first and add the test after" | Then it's not TDD, it's test-after. Delete the handler and start over with a failing test. |
| "A flat design looks unfinished; let me add a gradient / shadow / blur" | Flat paper IS the design. Chrome is 1px borders; depth is reserved for hover and floating layers. Adding glass or default shadows breaks the language. |
| "Any radius or background can be chosen per page" | Shape and light surface are project-level profiles from `frontend/design-profiles.md`. Controls consume the selected tokens; page-local theme and radius variants are forbidden. |
| "I'll just use the system font stack" | Plus Jakarta Sans + Geist Mono are the brand. Load them in `index.html`. |
| "Tailwind colors are close enough" | They are not. `#10b981` (Tailwind emerald) is not `#40a02b` (Catppuccin Green). The palette is harmonized. |
| "A quick `alert()` / inline `confirm()` is fine for now" | `errorBus` and `useConfirm` exist. Native browser dialogs are never shippable. |

## Red Flags - STOP and Re-read (both parts)

If you catch yourself doing any of these, stop and reload this skill:

Backend / scaffold:
- Writing `r.GET("/api/...", ...)` or `r.POST("/papi/...", ...)` directly in `router.go` or any non-generated file.
- Creating thrift files under `idl/` that `./script/hz_update.sh` does not run, or "anchor" IDLs whose declared routes are hand-registered elsewhere. Every thrift feeds `hz`; every `/api/*` / `/papi/*` route comes from hz output.
- Adding a `RegisterManual` / `manual.go` / any custom route-registration helper, or hand-editing `biz/router/register.go` in any way. `register.go` is never hand-edited - if registration looks wrong, re-run `./script/hz_update.sh` (see `backend/idl-workflow.md` "When the Generated Code Looks Wrong").
- Defining request/response structs inside a handler file (`type listFooReq struct ...`) instead of using the hz-generated `biz/model` structs.
- Hand-writing anything under `biz/model/` or `biz/dal/mysql/query/`. (`biz/dal/mysql/model/` IS hand-written - it is the schema source that `gorm/gen` consumes.)
- Skipping `hz` / `wire` / `cmd/generate` because the tool "is not installed" or "failed once". Install the pinned toolchain; codegen failure stops the line, it never licenses hand-writing generated code.
- Editing a file under `biz/router/api/` or `biz/router/papi/` other than `middleware.go`.
- Adding `gorm.io/gorm` calls in a handler. Handlers call services; services call `dal/mysql/query`.
- Reaching for `mysql.DB()`, `db.Model(...)`, string conditions (`Where("col = ?", v)`), map-based `Updates(...)`, or manual `.Offset()/.Limit()` in business code. All DB access uses the typed gen DAO per `backend/data-conventions.md`.
- Hand-wiring `container.Default = &Container{ UserService: user.NewService(...) }`. Use Wire.
- Storing the auth token in `localStorage`, or in any web storage under the default HttpOnly-cookie scheme (a sessionStorage bearer token exists only in header-auth deployments - see State Persistence).
- Calling `axios.get('/api/user/info')` directly in a component's `setup()` without gating on `authStore.user`.
- Adding `pinia-plugin-persistedstate` to persist auth state. Auth comes from server, never from disk.
- Putting Chinese / English string literals directly in a `.vue` template of a project that has vue-i18n. Use `t('key')` in page/business code; `zh.json` and `en.json` must have identical key sets. (The locked shared templates - `MainLayout` / `Pagination` / `Loading` / `BaseDialog` / `useConfirm` - carry their Chinese ops copy by design and are exempt.)
- Committing with VERSION updated by `build.sh` but the new VERSION not staged.
- Creating `dist.tar.zst` or `cmd/embedcompress` without first hitting the `frontend/dist/` size trigger documented in `backend/embed-strategy.md`.
- Creating any wails / offline_* file, or adding `//go:build offline` to anything.
- Using `createWebHistory()` instead of `createWebHashHistory()` without the user confirming a full-site SEO requirement. Hash mode is the default; history mode additionally requires a Go-side SPA fallback.
- Overriding Vite's output file naming to remove the content hash from built assets. The `assets/[name]-[hash]` fingerprint is mandatory.
- Importing `github.com/mattn/go-sqlite3`. It requires CGO and breaks static cross-compiles; the SQLite dialector is `github.com/glebarez/sqlite` (pure Go).
- Calling `goredis` anywhere outside `biz/extension/redis`. Call sites use the `IRedis` seam so the lightweight profile's `LocalRedis` keeps working.
- Caching an error or a not-found result, caching an open time window ("today" counters, in-progress aggregates), or filling a cache with the request context so a client disconnect kills the fill. Cache rules: `backend/data-conventions.md`.
- A manually triggered expensive operation (run/rebuild/export) with no global in-flight cap and no per-target cooldown, or a cap check separate from the spawn (count-then-act is a race).
- Putting business-rule validation only in the public HTTP handler while an admin endpoint, agent/tool API, or internal job writes the same data. Validation lives once in the service layer; per-entry-point validators drift into bypasses.
- Adding MySQL / Redis / Casbin dependencies to a project the user scoped as lightweight profile. Profiles are chosen with the user at scaffold time - do not silently upgrade; use `backend/lite-profile.md`.

Frontend / visual:
- Installing or importing any component library (Element Plus, Naive UI, Ant Design Vue, Vuetify, PrimeVue) or CSS framework (Tailwind, UnoCSS).
- Writing `#` followed by a hex color, or any rgba literal, in a component or page (everything is `var(--...)`; the only exceptions are SVG data-URI icons that cannot see CSS vars and user-data colors rendered from the DB).
- Writing a raw `border-radius` on a page/form button or input. Shared controls use `var(--radius-sm)`; choose `data-shape` once for the project instead of adding a local shape.
- Writing an off-scale `font-size` such as `12.5px` or a page-local pixel size. Use the `--text-*` scale from `tokens.css`.
- Writing `font-family:` with `-apple-system` before Plus Jakarta Sans.
- Writing `transition: all 0.2s ease` (must be spring + a `--duration-*` token over an explicit property list - color/transform/layout properties enumerated, never `all`; the shipped templates already use explicit property lists). Infinite looping animations (e.g. the spinner rotation) are exempt from the spring-easing rule and may use `linear`.
- Adding an ambient gradient, a grain/noise overlay, any decorative gradient, or any `backdrop-filter` (the system is flat paper; none of these exist).
- Giving a card, sidebar, or header a default box-shadow (shadows are for hover accents, floating layers, and the sticky action column only).
- A drawer in ANY direction (right/left/top/bottom slide-over). Small overlays are floating panels (the nav panel / `RowActionsMenu` pattern); forms and results go in `BaseDialog`.
- A raw numeric `z-index` (use the `--z-*` token ladder).
- Rendering an `h1` or a page header inside a page (the shell owns the page title).
- A nav item without an `icons.ts` icon, or a close button using a bare `x` glyph instead of `<Icon name="x">`.
- Making `.shell-layout`, `.shell-main`, or the window scroll; scroll belongs to `.page-stage` / `.split-master` / `.split-detail` only (sole exception: the template's built-in <=760px unlock).
- Styling a control in a page's `<style scoped>` when a base.css class or shared component already covers it.
- Redefining or specializing a shared base.css class in a page `<style>` block (no local `.muted-text` / `.pill` / `.detail-head` / `.split-detail` variants), or restating a class-owned value in an inline `style`.
- Micro-adjusting one control's alignment or size in page styles (magic `line-height` / `padding` / `margin` / `width`). Alignment bugs are fixed in the shared layer (`base.css` / the template upstream) so every control of that kind benefits; a per-control patch is a bug seed.
- A bare `<select>` with page-local arrow/background overrides (the global style already ships the data-URI chevron).
- A page-local date picker, calendar popup, or composite hour/minute field, or a page `<style>` rule / inline sizing on a `date` / `time` / `datetime-local` input (native inputs styled by base.css only; relative ranges are `<select>` presets).
- `table-layout: fixed`, per-cell ellipsis to squeeze columns, or a non-sticky action column.
- Hand-rolling a dialog, drawer, tabs keyboard nav, pagination, or status pill. Drawers are banned in every direction; do not reintroduce them.
- Reaching for Element Plus patterns (el-table, el-dialog, el-select); Element Plus is banned with no exceptions.
- A status label hardcoded in a page instead of registered in `lib/labels.ts`.
- Blank cells, `null`, or `undefined` rendered to the user (use `'—'` via the lib helpers), or a raw API sentinel number on screen (`-1` meaning unlimited renders `无限`, unknown counts render `未知`).
- English UI copy for user-facing text, or Chinese API field names.

All of these mean: pause, open this SKILL.md again, do the documented thing.

## File Index

Visual system (Part 2, files under `frontend/`):
- `frontend/design-profiles.md` - normative shape and light-surface choices, defaults, selection workflow, HTML attributes, and upgrade boundaries.
- `frontend/tokens.css` - complete CSS variables (profile selectors, light + `:root.dark`, self-hosted font declarations). Copied verbatim into the project.
- `frontend/base.css` - the global stylesheet that replaces a component library: element styles + all shared classes. Copied verbatim into the project.
- `frontend/code-editor.css` - highlight-overlay editor styles for `CodeEditor` / `JsonField`. Copied verbatim into the project.
- `frontend/assets/fonts/` - self-hosted Plus Jakarta Sans and Geist Mono WOFF2 files with SIL OFL licenses; copy to `public/fonts/`.
- `frontend/layouts/app-shell.md` - full shell contract: fixed frame, flat surfaces, icon sidebar with desktop collapse persistence, narrow-screen floating nav panel, shell page header, dark toggle, errorBus toast stack.
- `frontend/layouts/page-card.md` - standard non-split page anatomy (`.page-stage` scrolling, `.card`, `.section-head`, no page-level h1).
- `frontend/layouts/fill-and-scroll.md` - fill-viewport + internal-scroll flex pattern, aligned with `.split-master` / `.split-detail` terminology and `SplitResizer`.
- `frontend/components/buttons.md` - global button variants: default accent, `.muted`, `.danger`, `.btn-sm`, `.toggle-btn`, `.row-actions` rules.
- `frontend/components/cards.md` - `.card`, `.section-head`, `.detail-head` + `.detail-actions`, `.split-detail-empty`, `.edit-grid`.
- `frontend/components/forms.md` - global input/select/textarea/label styles (data-URI select chevron, dark variant), `.field-label`, `.required-mark`, `.check-inline`, focus ring.
- `frontend/components/chips.md` - `.pill` family (`status-*` / `sev-*` / `risk-*` / `pill-sm` / `pill-code`) + `StatusPill` / `SeverityPill` / `TierPill` + `labels.ts` as the only label registry.
- `frontend/components/feedback.md` - `Loading` with 200ms delay (inline vs overlay); `EmptyState` vs `TableEmptyRow` division of labor.
- `frontend/components/dialogs.md` - `BaseDialog`, `ConfirmDialog` + `useConfirm`, `dialogStack`, `RowActionsMenu`.
- `frontend/components/code.md` - `CodeEditor` vs `CodeViewer`, `JsonViewer`, `JsonField`, `MarkdownText`, `HoverPopover`.

Templates (Part 2, copy as a unit; customize only the `EDIT:` markers):
- `frontend/templates/MainLayout.vue` - the app shell.
- `frontend/templates/Icon.vue` + `frontend/templates/icons.ts` - the icon set.
- `frontend/templates/BaseDialog.vue` - dialog shell.
- `frontend/templates/ConfirmDialog.vue` + `frontend/templates/useConfirm.ts` - confirmation UI.
- `frontend/templates/dialogStack.ts` - modal stack.
- `frontend/templates/Loading.vue` - spinner (overlay + inline).
- `frontend/templates/EmptyState.vue` + `frontend/templates/TableEmptyRow.vue` - empty and error states.
- `frontend/templates/Pagination.vue` - pager with stale page-size normalization.
- `frontend/templates/SplitResizer.vue` - accessible split ratio drag, keyboard, reset, and persistence.
- `frontend/templates/useTabs.ts` - tabs keyboard navigation.
- `frontend/templates/useFormValidation.ts` - form error timing.
- `frontend/templates/errorBus.ts` - toast stack.
- `frontend/templates/useFetch.ts` + `frontend/templates/usePaginatedFetch.ts` - cancellable fetch wrappers with stale-response guards and optional timeouts.
- `frontend/templates/time.ts` - date/time formatters.
- `frontend/templates/errors.ts` - `errorMessage`.
- `frontend/templates/clipboard.ts` - `copyText`.

Scaffold references (Part 1; load on demand, not upfront):
- `backend/idl-workflow.md` - full thrift template, `hz_update.sh` body, middleware stub editing rules, paging convention, error envelope, generated drift checks, optional TypeScript contracts, manual route exception list.
- `backend/backend-skeleton.md` - main.go boot sequence in full, unified model registry, container/wire.go skeleton, mw.InitMw() body, config schema with all required keys, embed wiring.
- `frontend/skeleton.md` - vite config, axios refresh interceptor, Pinia auth store, vue-i18n setup, Vue Router lazy pattern, dual-entry build (public + admin), Prettier and `pnpm check`.
- `backend/auth-jwt-casbin.md` - JWT cookie scheme, refresh flow, Casbin RBAC model text + gorm-adapter DB-backed policy seed, RBAC middleware composition, explicit route-policy coverage, common pitfalls.
- `backend/release-flow.md` - build.sh annotated, release.sh annotated, Dockerfile, Caddyfile, docker-compose.yml, VERSION semantics, CI generation and quality gates.
- `backend/testing-conventions.md` - TDD cadence, table-driven Go test template, httptest pattern, isolated database integration, migration and transaction contracts, shared frontend behavior tests.
- `backend/data-conventions.md` - NORMATIVE gorm/gen + IRedis usage rules: typed DAO only, no string conditions/raw handles in business code, `WithContext` everywhere, `FindByPage` + `ClampList`, update/upsert/delete/transaction idioms; cache closed facts only (never errors or open windows), `ErrCacheMiss` vs backend failures, singleflight cold reads, fills that survive caller cancellation, key/TTL discipline, lock ordering, fail-closed guardrails.
- `backend/embed-strategy.md` - OPTIONAL. Plain `//go:embed` vs zstd-archive + runtime-extract. Trigger criteria, full compressed-embed `frontend_embed.go`, `cmd/embedcompress`, build.sh hook.
- `backend/lite-profile.md` - lightweight profile templates: `IRedis` seam + in-process `LocalRedis` with shared `ErrCacheMiss`, SQLite via `glebarez/sqlite` (pure Go), role-column `RequireRoleMw` replacing Casbin, app+caddy-only compose, upgrade path to the full profile.

## Quick Start (when this skill activates)

New project from zero:
```
1. Confirm MODULE_NAME and folder.
2. Choose infrastructure, routing, and design shape/surface profiles; defaults are documented in their decision sections.
3. Follow Part 1 "Mandatory Boot Sequence" steps 1-12 in order.
4. For each step requiring a template, open the corresponding backend/*.md or frontend/*.md.
5. Copy Universal Discipline and all chosen profile decisions into the new project's CLAUDE.md / AGENTS.md.
6. Before committing the initial scaffold, run ./build.sh and ./output/bootstrap.sh once.
```

Frontend page/component work (a seeded project):
```
1. Read the recorded project shape/surface; do not change it during ordinary feature work.
2. Ensure tokens.css + base.css + templates + self-hosted fonts are present; copy code-editor.css when needed.
3. For each page:
   pick the pattern in "Page Patterns"  -> split layout / page-stage / BaseDialog
4. For each control:
   read frontend/components/[type].md    -> use the global style or shared component
5. Verify against the "Red Flags" list -> invariant scan, selected light surface and dark theme
```
