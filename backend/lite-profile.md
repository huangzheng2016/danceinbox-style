# Lightweight Profile Reference

Applies when the user picks the **lightweight profile** at scaffold time (see SKILL.md "Infrastructure Footprint Decision"). Use this file INSTEAD of the MySQL/Redis/Casbin parts of `backend-skeleton.md` and `auth-jwt-casbin.md`. IDL-first, `hz` codegen, Wire, GORM gen, JWT cookie auth, i18n, and the release flow stay exactly as documented elsewhere.

The pattern: keep the full profile's seams (`IRedis`, the DAL init path) and swap in in-process providers behind them. The provider is fixed at scaffold time by the chosen profile - within the lightweight profile `NewRedis` is always local. There is no runtime provider switch: no config key picks the provider, and there are no `*_offline.go` file variants and no `//go:build offline` machinery.

## What changes

| Full profile | Lightweight profile |
|--------------|---------------------|
| MySQL 8 via `gorm.io/driver/mysql` | SQLite via `github.com/glebarez/sqlite` (pure Go, wraps modernc.org/sqlite) |
| go-redis client | In-process `LocalRedis` behind the same `IRedis` seam |
| Redsync distributed lock | `IRedis.Mutex(...)` (same interface, single process) |
| Casbin enforcer, DB-backed policy (optional upgrade in BOTH profiles) | `role` column on user table + `RequireRoleMw` (+ `ModeratorOrAdminMw`/`AdminRoleMw` sugar) — the default RBAC shared by both profiles |
| docker-compose: mysql + redis + app + caddy | docker-compose: app + caddy only |

## Rule: use glebarez, never mattn

`github.com/mattn/go-sqlite3` requires CGO. CGO breaks the static cross-compiled binary and the `debian:bookworm-slim` runtime story. The GORM dialector for SQLite is `github.com/glebarez/sqlite`, which sits on `modernc.org/sqlite` (pure Go). Same GORM API, same `gorm/gen` compatibility.

## biz/extension/redis - the IRedis seam

Every call site codes against `IRedis`. Nobody imports `goredis` outside this package.

`interface.go`:

```go
package redis

import (
	"context"
	"errors"
	"time"
)

var ErrCacheMiss = errors.New("cache miss")

type Locker interface {
	Lock() error
	TryLock() error
	Unlock() (bool, error)
}

type IRedis interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) bool
	Mutex(ctx context.Context, key string, ttl time.Duration) Locker
	TryLockWithKey(ctx context.Context, key string, ttl time.Duration) (Locker, error)
	LockWithKey(ctx context.Context, key string, ttl time.Duration) (Locker, error)
}
```

Note: the full profile's `IRedis` is a superset — it adds `GetClient`, `SubscribeChannel`, and `Publish` (see "Upgrade path" below); the lightweight profile drops them to keep `goredis` out of the dependency tree.

`local.go` (complete, copy-paste ready):

```go
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var localLockCounter atomic.Uint64

type localRedisEntry struct {
	data      []byte
	expiresAt time.Time
}

type localRedisLock struct {
	token     string
	expiresAt time.Time
}

type LocalRedis struct {
	mu    sync.Mutex
	items map[string]localRedisEntry
	locks map[string]localRedisLock
}

type localLocker struct {
	store *LocalRedis
	ctx   context.Context
	key   string
	ttl   time.Duration
	token string
}

func NewLocalRedis() *LocalRedis {
	return &LocalRedis{
		items: make(map[string]localRedisEntry),
		locks: make(map[string]localRedisLock),
	}
}

// NewRedis is the Wire provider. Lightweight profile: always local.
// Upgrade path: return a go-redis backed implementation here instead;
// no call site changes.
func NewRedis() IRedis {
	return NewLocalRedis()
}

func (r *LocalRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	entry := localRedisEntry{data: data}
	if expiration > 0 {
		entry.expiresAt = time.Now().Add(expiration)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[key] = entry
	return nil
}

func (r *LocalRedis) Get(ctx context.Context, key string, value interface{}) error {
	r.mu.Lock()
	entry, ok := r.items[key]
	if ok && entry.expired(time.Now()) {
		delete(r.items, key)
		ok = false
	}
	r.mu.Unlock()
	if !ok {
		return ErrCacheMiss
	}
	return json.Unmarshal(entry.data, value)
}

func (r *LocalRedis) Delete(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, key)
	return nil
}

func (r *LocalRedis) Exists(ctx context.Context, key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.items[key]
	if ok && entry.expired(time.Now()) {
		delete(r.items, key)
		return false
	}
	return ok
}

func (r *LocalRedis) Mutex(ctx context.Context, key string, ttl time.Duration) Locker {
	return &localLocker{store: r, ctx: ctx, key: key, ttl: ttl, token: newLocalLockToken()}
}

func (r *LocalRedis) TryLockWithKey(ctx context.Context, key string, ttl time.Duration) (Locker, error) {
	locker := &localLocker{store: r, ctx: ctx, key: key, ttl: ttl, token: newLocalLockToken()}
	if err := locker.TryLock(); err != nil {
		return nil, err
	}
	return locker, nil
}

func (r *LocalRedis) LockWithKey(ctx context.Context, key string, ttl time.Duration) (Locker, error) {
	locker := &localLocker{store: r, ctx: ctx, key: key, ttl: ttl, token: newLocalLockToken()}
	if err := locker.Lock(); err != nil {
		return nil, err
	}
	return locker, nil
}

func (e localRedisEntry) expired(now time.Time) bool {
	return !e.expiresAt.IsZero() && !now.Before(e.expiresAt)
}

func (l localRedisLock) expired(now time.Time) bool {
	return !l.expiresAt.IsZero() && !now.Before(l.expiresAt)
}

func (l *localLocker) Lock() error {
	deadline := time.Now().Add(3 * time.Second)
	for {
		if err := l.TryLock(); err == nil {
			return nil
		}
		select {
		case <-l.ctx.Done():
			return l.ctx.Err()
		default:
		}
		if time.Now().After(deadline) {
			return errors.New("lock already held")
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func (l *localLocker) TryLock() error {
	now := time.Now()
	expiresAt := time.Time{}
	if l.ttl > 0 {
		expiresAt = now.Add(l.ttl)
	}
	l.store.mu.Lock()
	defer l.store.mu.Unlock()
	if existing, ok := l.store.locks[l.key]; ok && !existing.expired(now) {
		return errors.New("lock already held")
	}
	l.store.locks[l.key] = localRedisLock{token: l.token, expiresAt: expiresAt}
	return nil
}

func (l *localLocker) Unlock() (bool, error) {
	l.store.mu.Lock()
	defer l.store.mu.Unlock()
	existing, ok := l.store.locks[l.key]
	if !ok || existing.token != l.token {
		return false, errors.New("lock is not held")
	}
	delete(l.store.locks, l.key)
	return true, nil
}

func newLocalLockToken() string {
	return fmt.Sprintf("%d:%d", time.Now().UnixNano(), localLockCounter.Add(1))
}
```

Semantics to preserve: values are JSON round-tripped (same as the go-redis impl), TTLs are honored, locks carry a token so `Unlock` cannot release someone else's lock. Both profiles return the same `ErrCacheMiss` sentinel for an absent or expired key. Call sites use `errors.Is(err, redis.ErrCacheMiss)` and never import `goredis`; other cache errors remain distinguishable for explicit degradation or fail-closed guardrails (see `data-conventions.md` "Cache Miss vs Backend Errors").

## biz/dal/mysql - SQLite via glebarez

The package name stays `dal/mysql` and the exported surface stays `InitMysql()` / `DB()` / `GetDBQuery()` / `Ping(ctx)` / `MaxLimit`. The differences from the full profile are confined to the DAL internals: the dialector, the `sqlite.*` config keys, the `MkdirAll` for the database file, the single-connection pool, and `Ping` inlined here instead of living in a separate `probe.go`. The switch to real MySQL is covered in "Upgrade path" below.

`init.go`:

```go
package mysql

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"<MODULE_NAME>/biz/dal/mysql/model"
	"<MODULE_NAME>/biz/dal/mysql/query"
	"<MODULE_NAME>/config"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	db   *gorm.DB
	q    *query.Query
	once sync.Once
)

func InitMysql() {
	tablePrefix := config.GetConfigStrOptional("sqlite.prefix")
	model.SetTablePrefix(tablePrefix)

	once.Do(func() {
		sqlitePath := strings.TrimSpace(config.GetConfigStrOptional("sqlite.path"))
		if sqlitePath == "" {
			sqlitePath = "./data/app.db"
		}
		if err := os.MkdirAll(filepath.Dir(sqlitePath), 0755); err != nil {
			panic(err)
		}

		var err error
		db, err = openSQLiteWithEnvLogger(sqlitePath)
		if err != nil {
			panic(err)
		}
		q = query.Use(db)

		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.SetMaxOpenConns(1) // sqlite: single writer, avoids "database is locked"
			sqlDB.SetMaxIdleConns(1)
			sqlDB.SetConnMaxLifetime(time.Hour)
		}

		log.Printf("[sqlite] init sqlite success: %s", sqlitePath)
		if err := autoMigrate(); err != nil {
			panic(err)
		}
	})
}

const MaxLimit = 50

// DB exposes the raw *gorm.DB for DAL-internal use ONLY (init, migrate,
// adapters built in this package). Business code goes through GetDBQuery().
func DB() *gorm.DB {
	return db
}

func GetDBQuery() *query.Query {
	return q
}

// Ping probes the database for the /healthz readiness check. It lives in the
// DAL so biz/handler/health.go never touches the raw handle. Same contract as
// the full profile's probe.go.
func Ping(ctx context.Context) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func openSQLiteWithEnvLogger(path string) (*gorm.DB, error) {
	dsn := path + "?cache=shared&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	if os.Getenv("DEBUG") != "" {
		return gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	}
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{})
}
```

Non-negotiables: WAL journal mode, `busy_timeout(5000)`, `foreign_keys(ON)`, `SetMaxOpenConns(1)`. Skipping any of these produces `database is locked` errors under concurrent requests.

`gorm/gen` note: the scaffold's `cmd/generate` uses `ApplyBasic(model.X{}, ...)` over the hand-written structs in `biz/dal/mysql/model/` — gen reads structs, not a live database, so it works identically for SQLite and MySQL (see `backend-skeleton.md` "cmd/generate"). Model structs stay dialect-agnostic (primary keys carry no explicit `type:` — see `backend-skeleton.md`), so moving to MySQL touches only the DAL wiring, not the models. Avoid DB-introspection gen modes (`GenerateAllTable`, `UseDB`) — those need a live DB of the right dialect and are how hand-written model dirs sneak back in.

## RBAC: role column + RequireRoleMw (scaffold-wide default)

This is not a lightweight-only fallback — it is the default RBAC for every project this skill seeds (see the RBAC stack-lock row). No Casbin enforcer, no policy table, no Redis pub/sub policy sync, unless policies must be edited at runtime. The three roles become a `role` column (`USER` / `MODERATOR` / `ADMIN`) on the user model, checked by middleware:

```go
// biz/mw/role.go
package mw

import (
	"context"
	"slices"

	"<MODULE_NAME>/biz/consts"

	"github.com/cloudwego/hertz/pkg/app"
)

// RequireRoleMw aborts with 403 unless the JWT-authenticated user holds
// one of the given roles. It is the default RBAC middleware in BOTH
// profiles; the Casbin enforcer middleware replaces it only after the
// optional Casbin escalation. Attach AFTER JwtAuthenticationMw.
func RequireRoleMw(roles ...string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if !slices.Contains(roles, GetJwtRole(ctx, c)) {
			c.AbortWithStatus(403)
			return
		}
		c.Next(ctx)
	}
}

// ModeratorOrAdminMw / AdminRoleMw are sugar over RequireRoleMw for the two
// common admin gates. They live here — NOT in casbin.go — so the default
// scaffold gates /papi/* (papi.rootMw uses ModeratorOrAdminMw) with no
// Casbin dependency. Both names keep working after the Casbin escalation;
// swap their call sites for enforcer-based middleware only when policies
// become runtime-editable.
func ModeratorOrAdminMw() app.HandlerFunc {
	return RequireRoleMw(consts.RoleModerator, consts.RoleAdmin)
}

func AdminRoleMw() app.HandlerFunc {
	return RequireRoleMw(consts.RoleAdmin)
}
```

The role comes from the JWT claim (`GetJwtRole`) — no DB hit per request. Consequence: a role change takes effect on the user's next login or token refresh, which is the accepted trade-off. Account disablement is still enforced per-request by `isJwtUserActive` inside `JwtParseMw` (see `auth-jwt-casbin.md`). (`CheckRoleAny` in `auth-jwt-casbin.md` reads the same claim, but it ships only with the Casbin upgrade — in the default scaffold, role checks go through `RequireRoleMw` and its sugar above.)

This covers the USER/MODERATOR/ADMIN model as long as policies are static (compile-time role lists per route). The moment policies must be edited at runtime, by non-developers, or per-resource-owner, that is the trigger to bring in Casbin per `auth-jwt-casbin.md` - treat it as profile upgrade, not a tweak.

## docker-compose (lightweight)

```yaml
services:
  app:
    build:
      context: .
      args:
        VERSION: ${VERSION:-dev}   # release.sh passes VERSION=x into the build
    restart: unless-stopped
    volumes:
      - ./data:/app/data                      # sqlite file: WORKDIR is /app, so the default sqlite.path "./data/app.db" lands on this mount
      - ./config/prod.json:/app/config/prod.json:ro
  caddy:
    image: caddy:2
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - ./data/caddy/data:/data
      - ./data/caddy/config:/config
    depends_on:
      - app
```

No mysql/redis services, no healthcheck dependencies on them, no `APP_CONFIG` env (config selection is the `-c prod` baked into the image CMD). Three alignment notes:

- `./data` must map to `/app/data`, NOT `/data` — the binary's working directory is the Dockerfile `WORKDIR /app`, so the relative default `./data/app.db` resolves to `/app/data/app.db`. A `/data` mount leaves the DB in the container's ephemeral layer, wiped by every `--force-recreate`.
- `release.sh` needs its infra bring-up parameterized for this profile: the stock script runs `docker compose up -d --remove-orphans mysql redis`, which fails with "no such service" here. `release-flow.md` exposes this as an `INFRA_SERVICES` variable — set it empty in the lite project. `INFRA_SERVICES` parameterizes ONLY the infra bring-up; it does not create or rename app services.
- The only supported deploy target is `prod`: this compose file defines just `app` + `caddy`, so `release.sh test` / `release.sh pre` fail with "no such service" (they build and recreate `app-test` / `app-pre`). If a lite project genuinely needs those environments, add the services and their `./config/<env>.json` mounts to this compose file first.

## Config keys (lightweight)

`config/local.json` and siblings carry:

```json
{
  "sqlite": { "path": "./data/app.db", "prefix": "" }
}
```

Drop the `mysql.*` and `redis.*` blocks entirely. `main.go` still calls `mysql.InitMysql()` and Wire still provides `redis.NewRedis` - the boot sequence does not change.

## Upgrade path (lightweight -> full)

- **Redis**: replace the `NewRedis` body with the full-profile go-redis implementation from `backend-skeleton.md` ("biz/extension/redis (full profile)"); its `IRedis` superset also adds `GetClient` / `SubscribeChannel` / `Publish`, which you need the moment Redsync or Casbin policy sync enters. No call site changes - that is what the seam is for.
- **MySQL**: provision MySQL, switch `biz/dal/mysql/init.go` to the full-profile version from `backend-skeleton.md`, and migrate data with a one-time script. This is more than a dialector swap: the full-profile `init.go` reads `mysql.*` config keys and a 100/10 connection pool, and it does NOT carry `Ping` — the full profile's `Ping` lives in a separate `probe.go`, so you must add that file too or `/healthz` stops compiling. `cmd/generate` needs no change (it reads structs, not a live DB). Do not promise automatic cross-dialect migration.
- **Casbin**: only if policies become runtime-editable — then follow `auth-jwt-casbin.md` wholesale (enforcer, gorm-adapter, `casbin_notify` pub/sub). Requires the full-profile Redis upgrade first (pub/sub). The role column can seed the initial policies.
