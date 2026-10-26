# MODULE_NAME

Hertz (Go) + Vue 3 app generated from the danceinbox-style skill. Profile: PROFILE (full: MySQL + Redis | lite: SQLite + in-process cache).

## Commands

```bash
./build.sh                        # frontend + binary -> output/
./output/run.sh                   # serve with config/local.json on :8080
cd frontend && pnpm dev           # Vite dev server, proxies /api and /papi to :8080
./script/hz_update.sh             # after editing idl/
go run ./cmd/generate             # after editing biz/dal/db/model
go generate ./container           # after adding a service
go test ./... && go vet ./...     # backend gate
cd frontend && pnpm check         # frontend gate
```

## How to add things

- **Endpoint or feature:** schema in `biz/dal/db/model` -> failing service test -> service -> wire -> IDL -> `hz_update.sh` -> fill handler -> `frontend/src/api` -> page. Copy the `example` feature at every step.
- **Management page (list, create, edit, delete):** copy `frontend/src/pages/ExampleList.vue` and `ExampleFormDialog.vue`.
- **Full detail page for one object:** copy `frontend/src/pages/ExampleDetail.vue` (route `meta.parent` gives the breadcrumb).
- **Overview page with charts:** copy `frontend/src/pages/Home.vue`.
- **Streamed output (agent runs, job logs):** backend `biz/service/run` (store events, then publish; plug a model into `run.Agent`) and a `/stream` IDL route; frontend `ExampleRuns.vue`.
- **File uploads:** `biz/service/attachment` and `ExampleAttachments.vue`.
- **Multi-step flow (import, setup):** copy `ExampleImport.vue` (`useWizard` + `Steps`).
- **Choosing a control** (select vs segmented vs combobox, alert vs toast, ...): the table at the top of each component in `frontend/src/base/`, or the skill's `frontend/design-system.md`.
- **Status or enum shown in the UI:** register it in `frontend/src/base/labels.ts`, render with `StatusPill`.
- **Nav entry:** `frontend/src/base/app.ts`.

## Rules

Backend
- Every `/api` and `/papi` route comes from `idl/` through hz. Never register those routes by hand, never edit generated files (`biz/router/**` except `middleware.go`, `biz/model/**`, `biz/dal/db/query/**`, `wire_gen.go`).
- Handlers bind, call one service method, convert, respond. Errors go through `resp.Fail`; client-visible errors are `errs.*` with a Chinese message.
- Services use only the typed query layer: no string conditions, no raw SQL, `UpdateSimple` for updates, preconditions in `Where` for state changes.
- Every route needs a session unless it is listed in `publicRoutes` (`biz/mw/auth.go`). Role checks: `mw.RequireRole`; ownership checks: in the service.
- New behavior and every bug fix start with a failing test. `go test ./...` passes with no services running.

Frontend
- `src/base/` and `src/styles/` are copied from the skill and not edited here (except `app.ts` and `labels.ts`). If something there is wrong, fix it in the skill and re-copy.
- No component library, no CSS framework, no hex colors or raw z-index in pages, no page styles for shared classes or native controls.
- One `.primary` per area; destructive actions confirm through `useConfirm`; every mutation ends in `toast.success` or `toast.error('<动作>失败', e)`.
- List state (filters, sort, page, selection, tab) lives in the URL via `useList` / `useRouteQuery`.
- Closing a form with edits asks first (`useDirtyGuard`); batch actions report done and skipped counts.
- Re-fetching while something runs uses `usePolling`, never `setInterval`. Markdown goes through `MarkdownText`, charts through `Chart`, never `v-html` or a second chart library.
- UI text is Chinese, written as plain statements. Times through `time.ts`, enums through `StatusPill`, empty values as `—`.

Code
- Comments say why, only where the code cannot. No comments that restate the code, point to docs, or narrate history ("legacy", "kept for compatibility", "reserved for future use").
- No commented-out code, no unused parameters, flags, hooks, or config keys "for later". Delete instead of deprecating.
- No fallbacks for states that cannot happen. Missing configuration fails at startup; it is not papered over with defaults.
- Match the surrounding code. Three similar lines beat a premature abstraction.
