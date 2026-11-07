# Backend Workflow

How the scaffold in `backend/scaffold/` is organized and how features are added to it. Every file referenced here exists in the scaffold; read it there rather than reconstructing it.

## Layout

```
idl/                      api.thrift, papi.thrift (service registries) + api/<feature>.thrift (DTOs)
biz/
  router/                 hz output. Only */middleware.go is hand-edited.
  model/                  hz output: request/response structs.
  handler/api, papi/      hz stubs with hand-filled bodies; convert.go maps DAL -> API models
  handler/resp/           Bind / Fail / Abort: the one error envelope
  handler/health.go       /livez, /healthz
  errs/                   errors the client may see (status + code + msg)
  actor/                  the signed-in user's id in the request context (set by mw.Auth)
  mw/                     Auth, RequireRole, RateLimit, AccessLog
  service/<feature>/      business logic over the typed query layer
  dal/db/model/           hand-written GORM structs: the schema
  dal/db/query/           gorm/gen output
  dal/db/                 Init, Query, Page, Contains; dbtest/ for tests
  extension/cache/        Cache interface + provider (Redis, or in-process in lite)
container/                Container struct, wire.go, wire_gen.go (generated)
config/                   config.go + <env>.example.json; real <env>.json are gitignored
cmd/generate/             go run ./cmd/generate
main.go router.go router_gen.go frontend_embed.go
```

Generated, never hand-edited: `biz/router/**` (except `middleware.go`), `biz/model/**`, `biz/dal/db/query/**`, `container/wire_gen.go`, `router_gen.go`. CI regenerates all of them and fails on any diff.

## The IDL rule

Every `/api/*` and `/papi/*` route is declared in the IDL and registered by hz. Never call `r.GET` / `r.POST` for those prefixes by hand, never hand-write a request struct in a handler, never edit generated routers. The only hand-registered routes are in `router.go`: `/livez`, `/healthz`, and the embedded frontend.

If hz fails, fix the toolchain or the IDL. Hand-writing generated code to get unblocked is how every drifted project started.

## Adding a feature

The order matters: each step produces what the next one consumes. The `example` feature is the reference for every step.

1. **Schema.** Add the struct to `biz/dal/db/model/` and to `model.All()`. Run `go run ./cmd/generate`.
2. **Service test first.** `biz/service/<feature>/service_test.go`, table-driven, against `dbtest.New(t)`. Watch it fail.
3. **Service.** `biz/service/<feature>/service.go`: `type Service struct{ q *query.Query; audit *audit.Service }`, `NewService(q, a)`. Return `errs.*` for anything the user should read. Record every change a person makes (see "Audit"). Follow `backend/data.md`.
4. **Wire.** Add `<feature>.NewService` to `container/wire.go` and a field to `Container`. Run `go generate ./container`.
5. **IDL.** DTOs in `idl/api/<feature>.thrift`, methods in the service block of `idl/api.thrift` (or `papi.thrift` for admin-only). Run `./script/hz_update.sh`.
6. **Handler.** Fill the stubs hz appended. Keep the signature and the `// @router` line.
7. **Frontend.** `src/api/<feature>.ts`, then pages copied from the reference pages (see `frontend/design-system.md`).

## IDL conventions

- One `<Feature>Service` per feature. Method names are globally unique verb-nouns (`ListExamples`, `SetExampleEnabled`) because all handlers of `api.thrift` land in one Go package.
- Paths are `/api/<feature>/<action>` in snake_case. `GET` for reads (query params), `POST` for writes (JSON body). No PUT/DELETE/path params; the generated binding stays uniform.
- Every response starts with `1: i32 code` and `2: string msg`; payload fields follow. Payload-free responses use `base.EmptyResp`, parameterless requests `base.EmptyReq`.
- Lists take `page` and `page_size` and return `items` and `total`. The service clamps with `db.Page`.
- int64 ids travel as JSON strings so JavaScript never rounds them: `1: i64 id (go.tag='json:"id,string"')` on every id field, request and response. This only replaces the json tag; query/form binding still works. The `,string` option does not apply to lists, so id lists are `list<string>` and parsed with `parseIDs` in the handler.
- Avoid field names that are keywords in any Thrift target language (`from`, `to`, `type`...): use `updated_from`, `kind`.
- Zero values that mean "none" (`ticket_id = 0`, `version = 0`, `created_at = 0`) are a contract the frontend must know: either make the field optional so absence is explicit, or document the sentinel next to the field so the page renders `—` / `未关联` instead of `#0`, `v0` or 1970.
- Validation that needs no database goes in `api.vd` annotations (`api.vd="len($) > 0"`); everything else in the service.

## Handlers

Every handler has the same shape:

```go
func GetExample(ctx context.Context, c *app.RequestContext) {
	var req example.GetExampleReq
	if !resp.Bind(c, &req) {
		return
	}
	row, err := container.Default.Example.Get(ctx, req.ID)
	if err != nil {
		resp.Fail(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, &example.ExampleResp{Item: toExample(row)})
}
```

- No business logic, no query calls, no status-code decisions. The handler binds, calls one service method, converts, and responds.
- `resp.Fail` turns an `*errs.Error` into its status and message and anything else into a logged 500 with the request id. Never write an error response by hand and never put `err.Error()` of an internal error in a response.
- Conversions DAL -> API and small handler helpers (`respondExample`, `ownerNames`, `parseIDs`) live in the hand-written `convert.go` of the handler package, never on the model types and never in the hz-generated handler files.

## Streaming endpoints (SSE)

- Still declared in the IDL (method returns `base.EmptyResp`, path ends in `/stream`) so auth, roles and rate limits apply as everywhere. The Caddyfile sends `*/stream` paths around compression with immediate flushing; other paths are compressed.
- The handler subscribes before writing anything, so errors before the stream starts are normal JSON errors; then `writeEvents` (`biz/handler/api/stream.go`) sends the replay, follows live events, heartbeats every 15 s, and ends after `done`.
- Producers store each event before publishing it (`biz/service/run`). Replays come from the store, so `Last-Event-ID` resumes exactly, and a run whose process died still ends with a `done` event.
- Expensive operations started by users have a per-target guard (one active run per example) and a global cap (4 concurrent runs), both reported as `errs.Conflict` with the reason. Runs execute with a detached context and their own timeout; `Cancel` stops them.
- `run.Agent` is the seam for a real model: stream tokens through `out.Text`, wrap each tool in `out.Tool`. `DemoAgent` keeps the path testable without one.

## Uploads

- One multipart route per owner (`/api/<feature>/attachment/upload`, `api.form` fields plus a `file` part). The service streams to a temp file while hashing and counting, enforces `upload.max_mb`, renames into `upload.dir` under a random name, and records name, size, sniffed type and SHA-256.
- The user's file name is display text only: never part of a path. Allowed extensions are a whitelist in the service. Downloads always send `Content-Disposition: attachment` and `nosniff`, so an uploaded HTML or SVG file can never run in the app's origin.
- Deleting removes the record, then the file, and succeeds if already gone. `upload.dir` must be a persistent volume in containers.

## Errors

`biz/errs` is the whole vocabulary: `Invalid` 400, `Unauthorized` 401, `Forbidden` 403, `NotFound` 404, `Conflict` 409. The message is Chinese, written for the person using the UI (`示例不存在`, `请先停用再删除`), because the frontend shows it verbatim in toasts.

- Say what is wrong in the user's terms, not the code's: `端口需为 1 到 65535 之间的整数`, not `port must be a safe integer`; `名称只能包含小写字母、数字和下划线`, not the regex. No field names, enum codes, HTTP statuses, or regexes in messages.
- Errors from another system (an RPC, a provider SDK, a mail server) are never passed through as-is. Map the cases the user can act on to an `errs.*` with a Chinese message; log the rest and let `resp.Fail` answer `服务器内部错误`. `rpc error: code = ... desc = ...` on a toast means a missing mapping.
- Failed binding answers `请求参数错误` and logs the binder's English detail; the frontend validates before sending, so users only see it when the client is wrong.

## Auth

- Sessions are two HttpOnly cookies: access (1h) and refresh (30d), both HS256 JWTs with only the user id and a type. Login and refresh issue both; refresh slides the 30-day window.
- `mw.Auth()` is in `rootMw()` of both `biz/router/api` and `biz/router/papi`, so every route requires a session by default. Routes that must work logged-out are listed in `publicRoutes` in `biz/mw/auth.go`. Adding a public endpoint means adding it there, deliberately.
- Every request loads the user from the database, so disabling a user or changing a role applies on the next request. `mw.CurrentUser(c)` returns it.
- Expired access + valid refresh answers 401 with `X-Token-Refresh-Needed: true`; the frontend refreshes once and retries. 401 without the header means no session; 403 always means "logged in but not allowed".
- Roles rank `USER < MODERATOR < ADMIN`. `/papi` requires MODERATOR in its `rootMw`; tighten a single route with `mw.RequireRole(model.RoleAdmin)` in its `_xxxMw`. Object-level rules (only the owner may edit) belong in the service, checked against `mw.CurrentUser`.
- Fields only some roles may see (notes, provider, region, internal addresses, cost) are emptied in `convert.go` for everyone else, based on `mw.CurrentUser`. Hiding them in the page is not access control: the response is visible in the browser's network panel.
- Passwords are bcrypt. Login returns the same error for an unknown user and a wrong password. `/api/auth/login` has its own strict rate limit in `_loginMw`.
- The first account: `PASSWORD=... ./output/run.sh -c local adduser admin ADMIN`.

## Audit

Every change a person makes through the UI leaves one `audit_logs` row, written by the service right after the change succeeds (`s.audit.Record(ctx, audit.Entry{...})`). The actor comes from the context (`biz/actor`, set by `mw.Auth`), so neither handlers nor services pass user ids around for it.

- One action per verb, named `<target>.<verb>` (`example.update`), declared as `model.Audit*` and labeled in the frontend's `AUDIT_ACTION`. A new target type also gets `model.Target*`, `AUDIT_TARGET`, and its field names in `AUDIT_FIELDS`.
- Updates store `Before` and `After` snapshots of the fields a person edits, as a small map keyed like the frontend's field labels, with related objects as names (`owner: "alice"`), not ids. No entry when nothing changed (saving an unchanged form, enabling something already enabled).
- Batch and import write one entry with a Chinese `Summary` (`批量停用 3 项，跳过 2 项`), not one per row.
- `TargetName` is the name at the time of the change, so the log still reads after a rename or delete.
- `Record` logs and swallows its own failure: the change already happened, and answering with an error would tell the user it did not.
- Who may read it: the full log is `/papi` (MODERATOR); an object's own history is an `/api` route next to the object (`ExampleHistory`), visible to whoever can see the object.

## Runtime settings

`biz/service/settings` is the reference for values an admin changes while the app runs: one typed `Settings` document, stored as append-only versions (`settings_versions`).

- Services read `settings.Current(ctx)` where they use a value (the run timeout when a run starts), so a save applies to the next request without a restart. Each field's comment names the service that reads it; a field nobody reads does not exist.
- A save carries the version it was edited from. If another version was saved since, the save answers 409 (`ErrConflict`); the unique index on `version` makes two concurrent saves from the same base fail the same way. Saving unchanged values returns the current version and writes nothing.
- Version 0 means never saved: `Current` returns `Defaults()`. Stored versions are decoded over the defaults, so a field added later gets its default in old versions.
- Restoring is a save of an old version's data with `restored_from` set, so history and the audit log say what happened and a restore can be undone.
- Settings vs config: if the process needs it at startup, it bounds the server itself (body size, pools, listen address), or it is a secret, it is config. Everything an admin should be able to change from the UI is a setting.
- Reading every request is one indexed query. Put a short cache in front of `Current` only when profiling says so, and invalidate it in `Save`.

## Config

`config/<env>.json`, selected with `-c <env>`, decoded into the typed `config.C` with unknown keys rejected and required keys validated at startup. Add a key by adding a struct field, a check in `validate()` if it is required, and the key to every `*.example.json`. Real configs are gitignored and live only on the machine that runs them.

## Testing

- `go test ./...` must pass on a clean clone with no services running. Service and middleware tests use `dbtest.New(t)` (temp SQLite, migrated); no mocks of the query layer.
- Table-driven tests; each case names the behavior (`"duplicate after trim"`), not the input.
- Test the rules, not the plumbing: validation, conflicts, state-dependent operations, zero-value updates, pagination edges, auth outcomes. A handler that only binds and delegates needs no test of its own.
- Write the failing test before the fix for every bug.
- Frontend: `pnpm check` runs format, lint, Vitest, typecheck, and build. Shared composables and reference pages ship with tests in `frontend/scaffold/src/test`.

## Toolchain and known cliffs

Install once: `go install github.com/cloudwego/hertz/cmd/hz@v0.9.7` and `go install github.com/cloudwego/thriftgo@v0.4.1` (hz shells out to thriftgo; without an explicit install it fetches an incompatible latest). Wire runs as a module tool (`go get -tool github.com/google/wire/cmd/wire`, `go tool wire`).

Resolve dependencies without versions at scaffold time, commit `go.mod`/`go.sum` and `pnpm-lock.yaml`, then upgrade one module at a time with build and tests. These pins are required:

| Module | Pin | Reason |
|---|---|---|
| `github.com/apache/thrift` | `v0.13.0` | thriftgo 0.4.1 output does not compile against newer thrift. `go mod tidy` floats it, so pin after every tidy that follows codegen. |
| `gorm.io/plugin/dbresolver` | `v1.6.2` | gorm/gen requires a version that does not compile against current gorm. |
| `typescript` (frontend) | `~6.0` | vue-tsc 3 cannot run on TypeScript 7. |

`hz`, `thriftgo`, and `apache/thrift` move together; bump them in one deliberate change.
