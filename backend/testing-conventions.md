# Testing Conventions Reference

The rules for tests in a project seeded by this skill: TDD cadence, table-driven Go tests, httptest, database integration, and Vitest for shared frontend behavior. Read this before writing the first test.

## The Floor (what every project enforces)

- TDD is the default for any feature or bugfix. Write the failing test first, watch it fail, write the minimal implementation, watch it pass.
- Every `biz/service/<feature>` package has a `service_test.go` covering each public method's happy path AND at least one refusal path. Multi-write transactions include a second-write failure test that proves the first write rolled back.
- Every reusable backend middleware has a unit test. JWT, Casbin, audit, rate-limit. Without these, regressions silently weaken auth.
- Model and migration contracts cover every `MigratedModels()` entry having a generated DAO, additive migration on a fresh and old schema, a second no-op run, and rejection of invalid definitions or dirty data. Soft-delete models prove deleted rows are hidden, unique names can be recreated, and tombstones remain.
- Generated-source CI reruns hz, GORM gen, and Wire on a clean checkout and fails on drift; it also runs `go vet ./...` and a gofmt zero-diff check.
- `go test ./...` must pass on a clean clone. CI runs that exact command. Unit tests stub third-party services; dialect- or migration-sensitive integration tests use isolated real database instances gated by their DSN and skip cleanly when unavailable.

## What is NOT mandatory (the allowed shortcut)

- Vue component snapshot tests. Shared async and interaction behavior tests are still required as described below.
- E2E tests via Playwright. If you do add E2E, scope it to login + one critical user flow, not every page.
- 100% coverage. Aim for "every branch that can fail in prod has at least one test that triggers it."

## TDD Cadence

For any non-trivial change:

```
1. Identify the smallest behavior to add.
2. Write the test, asserting the behavior. Run -> RED.
3. Write the minimal code to pass. Run -> GREEN.
4. Refactor without changing behavior. Run -> GREEN.
5. Commit.
```

If a change is truly mechanical (rename a variable, reformat), tests not required for that single change. But the moment branching or external state enters, you write the test first.

If you find yourself writing the implementation and then "I'll add the test after" — stop. Delete the implementation, write the test, watch it fail, then re-implement. Test-after produces tests that ratify whatever the code happens to do, not what it should do.

## Table-Driven Service Test

`biz/service/<feature>/service_test.go`:

```go
package user_test

import (
    "context"
    "errors"
    "testing"

    "<MODULE_NAME>/biz/dal/mysql/model"
    "<MODULE_NAME>/biz/dal/mysql/query"
    "<MODULE_NAME>/biz/dal/mysql/testutil"
    "<MODULE_NAME>/biz/service/user"
)

func TestServiceImpl_GetByID(t *testing.T) {
    cases := []struct {
        name     string
        id       int64
        seed     []*model.User
        wantErr  error
        wantName string
    }{
        {
            name:     "found",
            id:       1,
            seed:     []*model.User{{ID: 1, Username: "alice"}},
            wantName: "alice",
        },
        {
            name:    "not found",
            id:      99,
            seed:    nil,
            wantErr: user.ErrNotFound,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            q := query.Use(testutil.MustDB(t)) // fresh disposable schema per case
            if len(tc.seed) > 0 {
                if err := q.User.WithContext(context.Background()).Create(tc.seed...); err != nil {
                    t.Fatalf("seed: %v", err)
                }
            }
            svc := user.NewWithQuery(q)
            got, err := svc.GetByID(context.Background(), tc.id)

            if !errors.Is(err, tc.wantErr) {
                t.Fatalf("err = %v, want %v", err, tc.wantErr)
            }
            if tc.wantErr == nil && got.Username != tc.wantName {
                t.Fatalf("name = %q, want %q", got.Username, tc.wantName)
            }
        })
    }
}
```

Conventions:
- One `t.Run(tc.name, ...)` per case for failure clarity.
- Inputs and expected output as STRUCT FIELDS, not positional args.
- `errors.Is(err, sentinel)` for sentinel comparisons. `errors.As` for wrapped types.
- No `assert.X` libraries by default. `t.Fatalf` is enough. Add `testify` only if you need rich diffs.
- Seed data is part of the case definition. No shared global fixtures across cases.

## Service Tests: Real DB via a Constructor Seam

The typed gen layer has NO mocks — `cmd/generate` does not emit interfaces, and hand-written fake DALs drift from real MySQL/SQLite behavior. So:

- Pure helpers (validation, mapping, pricing, parsing) are plain functions — unit-test them without any DB.
- Anything touching `query` runs against a disposable database (`testutil.MustDB` below), with the service built from a test query object. Give every service a second constructor for this:

```go
// biz/service/user/service.go
func NewService() Service { return &serviceImpl{q: mysql.GetDBQuery()} } // Wire provider (prod)

// NewWithQuery is the test seam. Same struct, caller-supplied query root.
func NewWithQuery(q *query.Query) Service { return &serviceImpl{q: q} }
```

```go
// biz/service/user/service_test.go
func TestUserService_GetByID(t *testing.T) {
    svc := user.NewWithQuery(query.Use(testutil.MustDB(t)))
    _, err := svc.GetByID(context.Background(), 42)
    if !errors.Is(err, user.ErrNotFound) {
        t.Fatalf("err = %v, want user.ErrNotFound", err)
    }
}
```

The service translates `gorm.ErrRecordNotFound` into its own package sentinel (`user.ErrNotFound`) before returning — tests and handlers assert the sentinel, never the gorm error (rule: `data-conventions.md` "Reads").

`NewService` is the production constructor (Wire); `NewWithQuery` exists for tests. Do not invent fake DAL interfaces — they re-introduce the mock/prod divergence this doc forbids.

## httptest for Handlers

`biz/handler/api/user_service_test.go`:

```go
package api_test

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "testing"

    "<MODULE_NAME>/biz/dal/mysql/model"
    "<MODULE_NAME>/biz/dal/mysql/query"
    "<MODULE_NAME>/biz/dal/mysql/testutil"
    "<MODULE_NAME>/biz/handler/api"
    "<MODULE_NAME>/biz/service/user"
    "<MODULE_NAME>/container"
    "github.com/cloudwego/hertz/pkg/app/server"
    "github.com/cloudwego/hertz/pkg/common/ut"
    "golang.org/x/crypto/bcrypt"
)

func TestUserLogin_HappyPath(t *testing.T) {
    // Handlers read their services from container.Default — seed it BEFORE hitting any route.
    q := query.Use(testutil.MustDB(t))
    hash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
    if err != nil {
        t.Fatalf("hash: %v", err)
    }
    if err := q.User.WithContext(context.Background()).Create(&model.User{Username: "alice", PasswordHash: string(hash)}); err != nil {
        t.Fatalf("seed: %v", err)
    }
    container.Default = &container.Container{
        UserService: user.NewWithQuery(q),
    }

    h := server.New(server.WithHostPorts("127.0.0.1:0"))
    // Test-only throwaway server instance. Hand-registering a route is legal HERE ONLY.
    // In app code this route MUST come from hz via the IDL - never hand-register.
    h.POST("/api/user/login", api.UserLogin)

    body, _ := json.Marshal(map[string]string{"identifier": "alice", "password": "correct"})
    w := ut.PerformRequest(h.Engine, "POST", "/api/user/login",
        &ut.Body{Body: bytes.NewReader(body), Len: len(body)},
        ut.Header{Key: "Content-Type", Value: "application/json"})

    if w.Code != http.StatusOK {
        t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
    }
}
```

Use httptest for routing + middleware-layer tests. For pure logic, prefer service-layer tests (faster). Handler tests MUST seed `container.Default` with a test-backed service instance first — handlers resolve their dependencies through the container, so an unseeded container nil-panics inside the handler.

## MySQL Integration Tests

DO NOT mock MySQL with `sqlmock` for paths that exercise schema or migrations. Use a real disposable DB.

`biz/dal/mysql/testutil/db.go`:

```go
package testutil

import (
    "fmt"
    "os"
    "testing"
    "time"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

// MustDB returns a gorm.DB pointed at a fresh test schema.
// Set TEST_MYSQL_DSN=root:change_me_root_password@tcp(127.0.0.1:3306)/
// (no query params - they are appended per-connection below).
func MustDB(t *testing.T) *gorm.DB {
    t.Helper()
    dsn := os.Getenv("TEST_MYSQL_DSN")
    if dsn == "" {
        t.Skip("TEST_MYSQL_DSN not set; skipping integration test")
    }
    schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
    root, err := gorm.Open(mysql.Open(dsn))
    if err != nil { t.Fatalf("connect root: %v", err) }
    if err := root.Exec("CREATE DATABASE " + schema + " CHARACTER SET utf8mb4").Error; err != nil {
        t.Fatalf("create schema: %v", err)
    }
    t.Cleanup(func() {
        _ = root.Exec("DROP DATABASE " + schema).Error
    })

    db, err := gorm.Open(mysql.Open(dsn + schema + "?parseTime=True&loc=Local"))
    if err != nil { t.Fatalf("connect schema: %v", err) }

    // Run migrations.
    if err := db.AutoMigrate(/* models */); err != nil { t.Fatalf("migrate: %v", err) }
    return db
}
```

Usage:
```go
func TestQueryUser(t *testing.T) {
    db := testutil.MustDB(t)
    // ... seed, query, assert.
}
```

Integration tests that need MySQL run only when `TEST_MYSQL_DSN` is set. CI exposes that variable from a docker service; local devs skip them unless they `docker compose up mysql` first.

## Lightweight profile: SQLite test DB

Same `MustDB(t)` signature, no env var, no docker — sqlite IS the production dialect, so these tests always run:

```go
package testutil

import (
    "path/filepath"
    "testing"

    "github.com/glebarez/sqlite"
    "gorm.io/gorm"
)

func MustDB(t *testing.T) *gorm.DB {
    t.Helper()
    dsn := filepath.Join(t.TempDir(), "test.db") + "?cache=shared&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
    db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
    if err != nil { t.Fatalf("open sqlite: %v", err) }
    sqlDB, err := db.DB()
    if err == nil {
        sqlDB.SetMaxOpenConns(1)
        t.Cleanup(func() { sqlDB.Close() })
    }
    if err := db.AutoMigrate(/* models */); err != nil { t.Fatalf("migrate: %v", err) }
    return db
}
```

`go test ./...` in a lightweight project needs no running services.

## Middleware Tests

`biz/mw/jwt_test.go` (pattern):

```go
func TestJwtAuthenticationMw(t *testing.T) {
    cases := []struct {
        name       string
        status     int
        wantStatus int
        wantHeader string
    }{
        {"success", JWT_STATUS_SUCCESS, 0, ""},
        {"refresh", JWT_STATUS_REFRESH, 403, "true"},
        {"fail",    JWT_STATUS_FAIL,    401, ""},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            // github.com/cloudwego/hertz/pkg/common/ut — the real signature is
            // CreateUtRequestContext(method, url string, body *ut.Body, headers ...ut.Header).
            c := ut.CreateUtRequestContext("GET", "/", nil)
            c.Set(JwtStatus, tc.status)
            JwtAuthenticationMw()(context.Background(), c)
            if tc.wantStatus != 0 && c.Response.StatusCode() != tc.wantStatus {
                t.Fatalf("status = %d, want %d", c.Response.StatusCode(), tc.wantStatus)
            }
            if got := string(c.Response.Header.Get(consts.HeaderRefreshNeed)); got != tc.wantHeader {
                t.Fatalf("header = %q, want %q", got, tc.wantHeader)
            }
        })
    }
}
```

## Frontend: Vitest

Shared async logic and non-trivial interaction contracts need behavior tests. Page-level snapshots remain optional. Cover the reusable behavior in this order:
1. Pinia stores (state transitions, side effects).
2. The axios interceptor (refresh logic, retry).
3. Route guards (auth gating).
4. Composables that orchestrate API + state: `useFetch` / `usePaginatedFetch` must cover cancellation, stale responses, timeout, and scope disposal; URL state must cover browser back/forward, sibling query keys, filter-driven page reset, and deep-link selection.
5. Shared interaction components: `SplitResizer` keyboard/bounds/reset/persistence, `TableEmptyRow` error-vs-empty/retry, and `Pagination` stale page-size normalization.

Don't snapshot every `.vue`. Snapshots rot and nobody reads diffs. Test the behavior a caller relies on, not CSS pixels or framework rendering.

Example, `src/stores/auth.test.ts`:

```ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from './auth'

vi.mock('@/api', () => ({
  getUserInfo: vi.fn(async () => ({ code: 0, user: { id: 1, username: 'alice', email: 'a@b.c', role: 'USER' } })),
  userLogin:   vi.fn(),
  userLogout:  vi.fn(),
}))

describe('useAuthStore', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('init populates user', async () => {
    const store = useAuthStore()
    await store.init()
    expect(store.user?.username).toBe('alice')
    expect(store.loading).toBe(false)
    expect(store.isAuthed).toBe(true)
  })

  it('init handles failure as logged out', async () => {
    const { getUserInfo } = await import('@/api')
    ;(getUserInfo as any).mockRejectedValueOnce(new Error('500'))
    const store = useAuthStore()
    await store.init()
    expect(store.user).toBeNull()
    expect(store.isAuthed).toBe(false)
  })
})
```

Run `pnpm test` for the focused loop and `pnpm check` before delivery (test + zero-warning lint + format check + build with typecheck). The scaffold keeps `--passWithNoTests` only so a brand-new empty frontend can boot; it does not waive the shared behavior tests required above.

## What NOT to Test

- The Go standard library or third-party libs. Test YOUR code's use of them.
- `hz`-generated router files. They are mechanical.
- Pinia / Vue Router / Vite themselves.
- The fact that `t('foo')` returns a string when called. Test the i18n KEY consistency at lint time (a small `scripts/check-i18n-parity.mjs` is more useful).
- "Does Vue render this template" — covered by the framework.

## Red Flags

| Symptom | What it means |
|---------|--------------|
| Test reads from production-shaped DSN | You're testing prod's database. Stop. Use `TEST_MYSQL_DSN`. |
| `sqlmock.ExpectQuery(...)` against a join query | You will pass the test even when the real query is wrong. Use real MySQL. |
| Test only checks `err == nil` after a mutation | You did not verify state changed. Re-read the row and assert. |
| One mega test with 20+ assertions | Split per case. The next reader will skim and miss something. |
| Test that needs `time.Sleep(100*time.Millisecond)` | Race. Replace with a synchronization primitive or condition check. |
| Test that touches the network (real OpenAI, real SMTP) | Stub via the extension interface. |
| `if testing.Short() { t.Skip() }` everywhere | You have a test suite that is too slow. Profile, don't gate. |
| Frontend snapshot diffs in every PR | Snapshots are noise. Replace with behavioral assertions or delete. |

## CI Wiring

Recommend two CI jobs:
1. Fast: `go test ./...` without `TEST_MYSQL_DSN`. Runs unit + pure-function tests; DB tests self-skip. Should be under 1 minute.
2. Integration: `go test ./...` with `TEST_MYSQL_DSN` set (full profile; lightweight profile needs no services), plus `pnpm run test` if you added frontend tests. Allowed to be slower.

PR gate is job 1. Job 2 runs on merge to main + nightly.

## Single-Command Smoke

Every project this scaffold seeds must satisfy:

```bash
go test ./... && (cd frontend && pnpm run lint && pnpm run test)
```

If that command works, the project is in green state. Document this in `README.md` under "Tests". The `pnpm run test` step is safe with zero frontend tests (`--passWithNoTests`, see "Vitest" above).
