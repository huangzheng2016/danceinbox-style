# Auth, JWT, Casbin Reference

Complete mechanics for the cookie-based JWT + Casbin RBAC stack. Read this before touching `biz/mw/jwt.go`, `biz/mw/casbin.go`, or any login/refresh handler.

## Token Scheme

| Token | Cookie name | TTL | Cookie attrs |
|-------|------------|-----|--------------|
| Access | `ACCESS_TOKEN` | 3600s (1 hour) | HttpOnly, Secure per `jwt.cookie_secure`, SameSite=Lax, Path=`/` |
| Refresh | `REFRESH_TOKEN` | 2592000s (30 days) | HttpOnly, Secure per `jwt.cookie_secure`, SameSite=Lax, Path=`/` |

Set `jwt.cookie_secure=true` whenever the deployment is reached over TLS, including behind a TLS-terminating proxy (`server.tls` is only for the app terminating TLS itself). TTLs above are the defaults; the effective values come from `jwt.access_ttl_seconds` / `jwt.refresh_ttl_seconds` in config (one source for both signing and cookie Max-Age — see `AccessTTL`/`RefreshTTL` below). Both cookies use `Path=/` so `ClearAuthCookies` can clear them with the same path.

Both tokens are signed with the same HS256 secret (`jwt.secret` in config). Claims:
```json
{
  "uid": 12345,
  "role": "USER",
  "type": "access",        // or "refresh"
  "iat": 1735000000,
  "exp": 1735003600
}
```

`type` is the discriminator. The same secret signs both; never use one in the other's slot.

## Flow

```
Login
  POST /api/user/login  -> verify password
                        -> issue access + refresh
                        -> Set-Cookie: ACCESS_TOKEN=...; REFRESH_TOKEN=...
                        -> respond { code:0, user:{...} }

Normal request
  GET /api/whatever
    Cookie: ACCESS_TOKEN=...; REFRESH_TOKEN=...
  -> JwtParseMw decodes ACCESS_TOKEN
    -> valid: status = JWT_STATUS_SUCCESS, ctx has uid + role
    -> handler runs

Expired access
  GET /api/whatever
    Cookie: ACCESS_TOKEN=<expired>; REFRESH_TOKEN=<valid>
  -> JwtParseMw: access fails to verify
                 refresh verifies
                 status = JWT_STATUS_REFRESH
  -> JwtAuthenticationMw: refuse with 403
                          set X-Token-Refresh-Needed: true
  -> client (axios interceptor) sees 403 + header
  -> GET /api/user/refresh
                          -> RefreshJwt mints new ACCESS_TOKEN
                             (the refresh token itself is NOT rotated)
                          -> Set-Cookie: ACCESS_TOKEN=<new>; REFRESH_TOKEN=<same value, re-sent>
                          -> respond 200
  -> client retries original GET /api/whatever
```

Refresh re-sends BOTH cookies: a fresh access token plus the unchanged refresh token. Re-issuing the refresh cookie resets its Max-Age, so an active user's session slides forward — a deliberate sliding 30-day window, not token rotation. A user idle for 30 days still expires.

## biz/consts/auth.go

```go
package consts

const (
    AccessTokenCookie  = "ACCESS_TOKEN"
    RefreshTokenCookie = "REFRESH_TOKEN"
    HeaderRefreshNeed  = "X-Token-Refresh-Needed"

    JwtTypeAccess  = "access"
    JwtTypeRefresh = "refresh"

    RoleUser      = "USER"
    RoleModerator = "MODERATOR"
    RoleAdmin     = "ADMIN"
)
```

## Password hashing (bcrypt, no exceptions)

User passwords are stored as `bcrypt` hashes via `golang.org/x/crypto/bcrypt`. Never unsalted SHA-256 / MD5 / plain hex — not even "for the demo": seeds live in git and habits stick.

```go
// creating / changing a password (service layer):
hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)

// verifying at login (service layer):
if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(plain)); err != nil {
    return nil, ErrBadCredentials
}
```

## biz/mw/jwt.go (skeleton)

```go
package mw

import (
    "context"
    "errors"
    "strings"
    "time"

    "<MODULE_NAME>/biz/consts"
    "<MODULE_NAME>/biz/dal/mysql"
    "<MODULE_NAME>/config"

    "github.com/cloudwego/hertz/pkg/app"
    "github.com/cloudwego/hertz/pkg/protocol"
    "github.com/golang-jwt/jwt/v5"
)

const (
    JwtStatus = "_jwt_status"
    JwtUID    = "_jwt_uid"
    JwtRole   = "_jwt_role"
)

const (
    JWT_STATUS_FAIL    = 0
    JWT_STATUS_SUCCESS = 1
    JWT_STATUS_REFRESH = 2
)

var jwtSecret []byte

func initJwtKeys() {
    jwtSecret = []byte(config.GetConfigStr("jwt.secret"))
}

func parseToken(raw string) (jwt.MapClaims, error) {
    tok, err := jwt.Parse(strings.TrimSpace(raw), func(_ *jwt.Token) (any, error) {
        return jwtSecret, nil
    }, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
    if err != nil {
        return nil, err
    }
    claims, ok := tok.Claims.(jwt.MapClaims)
    if !ok {
        return nil, errors.New("not map claims")
    }
    return claims, nil
}

func claimType(claims jwt.MapClaims) string {
    v, _ := claims["type"].(string)
    return v
}

func JwtParseMw() app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        c.Set(JwtStatus, JWT_STATUS_FAIL)

        access := string(c.Cookie(consts.AccessTokenCookie))
        if access != "" {
            if claims, err := parseToken(access); err == nil && claimType(claims) == consts.JwtTypeAccess {
                if active, err := isJwtUserActive(ctx, claims); err == nil && active {
                    c.Set(JwtStatus, JWT_STATUS_SUCCESS)
                    c.Set(JwtUID, int64FromClaim(claims, "uid"))
                    c.Set(JwtRole, stringFromClaim(claims, "role"))
                    c.Next(ctx)
                    return
                }
            }
        }

        refresh := string(c.Cookie(consts.RefreshTokenCookie))
        if refresh != "" {
            if claims, err := parseToken(refresh); err == nil && claimType(claims) == consts.JwtTypeRefresh {
                if active, err := isJwtUserActive(ctx, claims); err == nil && active {
                    c.Set(JwtStatus, JWT_STATUS_REFRESH)
                    c.Set(JwtUID, int64FromClaim(claims, "uid"))
                    c.Set(JwtRole, stringFromClaim(claims, "role"))
                }
            }
        }
        c.Next(ctx)
    }
}

func JwtAuthenticationMw() app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        status, _ := c.Get(JwtStatus)
        switch status {
        case JWT_STATUS_SUCCESS:
            c.Next(ctx)
        case JWT_STATUS_REFRESH:
            c.Header(consts.HeaderRefreshNeed, "true")
            c.AbortWithStatus(403)
        default:
            c.AbortWithStatus(401)
        }
    }
}

func RefreshJwt(ctx context.Context, c *app.RequestContext) (newAccess string, newRefresh string, err error) {
    refresh := strings.TrimSpace(string(c.Cookie(consts.RefreshTokenCookie)))
    claims, err := parseToken(refresh)
    if err != nil || claimType(claims) != consts.JwtTypeRefresh {
        return "", "", errors.New("invalid refresh")
    }
    if active, err := isJwtUserActive(ctx, claims); err != nil || !active {
        return "", "", errors.New("user not active")
    }
    uid := int64FromClaim(claims, "uid")
    role := stringFromClaim(claims, "role")
    newAccess, err = MintToken(uid, role, consts.JwtTypeAccess, AccessTTL())
    if err != nil {
        return "", "", err
    }
    return newAccess, refresh, nil
}

// AccessTTL / RefreshTTL are the single source for token lifetimes: MintToken
// signs with them and SetAuthCookies writes the same values as cookie Max-Age.
// Both read the same config keys, so a cookie never outlives its token (or the
// reverse) when ops changes the TTLs.
func AccessTTL() time.Duration {
    ttl := config.GetConfigInt("jwt.access_ttl_seconds")
    if ttl <= 0 {
        ttl = 3600
    }
    return time.Duration(ttl) * time.Second
}

func RefreshTTL() time.Duration {
    ttl := config.GetConfigInt("jwt.refresh_ttl_seconds")
    if ttl <= 0 {
        ttl = 30 * 24 * 3600
    }
    return time.Duration(ttl) * time.Second
}

func MintToken(uid int64, role, typ string, ttl time.Duration) (string, error) {
    claims := jwt.MapClaims{
        "uid":  uid,
        "role": role,
        "type": typ,
        "iat":  time.Now().Unix(),
        "exp":  time.Now().Add(ttl).Unix(),
    }
    return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

func SetAuthCookies(c *app.RequestContext, access, refresh string) {
    // cookieSecure rule: see the Token Scheme table above / "config Schema" in
    // backend-skeleton.md. Never reuse server.tls for it.
    cookieSecure := config.GetConfigBool("jwt.cookie_secure")
    sameSite := protocol.CookieSameSiteLaxMode
    c.SetCookie(consts.AccessTokenCookie, access, int(AccessTTL().Seconds()), "/", config.GetConfigStrOptional("jwt.cookie_domain"), sameSite, cookieSecure, true)
    c.SetCookie(consts.RefreshTokenCookie, refresh, int(RefreshTTL().Seconds()), "/", config.GetConfigStrOptional("jwt.cookie_domain"), sameSite, cookieSecure, true)
}

func ClearAuthCookies(c *app.RequestContext) {
    c.SetCookie(consts.AccessTokenCookie, "", -1, "/", config.GetConfigStrOptional("jwt.cookie_domain"), protocol.CookieSameSiteLaxMode, false, true)
    c.SetCookie(consts.RefreshTokenCookie, "", -1, "/", config.GetConfigStrOptional("jwt.cookie_domain"), protocol.CookieSameSiteLaxMode, false, true)
}

func isJwtUserActive(ctx context.Context, claims jwt.MapClaims) (bool, error) {
    uid := int64FromClaim(claims, "uid")
    q := mysql.GetDBQuery().User
    u, err := q.WithContext(ctx).Where(q.ID.Eq(uid)).First()
    if err != nil {
        return false, err // DENY on ANY error, including ErrRecordNotFound.
    }
    return u.Active == 1, nil // requires the `Active int32` column on the user model
}

func GetJwtId(_ context.Context, c *app.RequestContext) int64 {
    v, ok := c.Get(JwtUID)
    if !ok { return 0 }
    return v.(int64)
}

func GetJwtRole(_ context.Context, c *app.RequestContext) string {
    v, ok := c.Get(JwtRole)
    if !ok { return "" }
    return v.(string)
}

// helpers
func int64FromClaim(c jwt.MapClaims, k string) int64 {
    switch v := c[k].(type) {
    case float64: return int64(v)
    case int64:   return v
    case int:     return int64(v)
    }
    return 0
}
func stringFromClaim(c jwt.MapClaims, k string) string { v, _ := c[k].(string); return v }
```

**Non-obvious choices and why**:
- `parseToken` pins the accepted algorithms via `jwt.WithValidMethods`. Without it the parser accepts any `alg` the library supports — an attacker who can present a token signed with a different algorithm (or the `none` variant in older libs) gets signature checks that were never meant for this deployment. The mint side only ever emits HS256, so the parse side allows exactly HS256.
- `parseToken` trims whitespace because some proxies append `; ` to cookie values.
- `claimType` reads via string-coercion against `jwt.MapClaims` to survive type-juggled claim shapes from old libs.
- `JwtParseMw` does NOT set 401/403 — it only annotates the context. `JwtAuthenticationMw` is what enforces. This split lets public routes (e.g. `/api/public/list`) still read uid when present without forcing auth.

## biz/mw/casbin.go (DB-backed + Redis notify)

```go
package mw

import (
    "context"
    "log"

    "<MODULE_NAME>/biz/dal/mysql"
    "<MODULE_NAME>/container"
    "github.com/casbin/casbin/v2"
    casbinmodel "github.com/casbin/casbin/v2/model"
    "github.com/cloudwego/hertz/pkg/app"
)

var enforcer *casbin.Enforcer

// rbacModelText lives in code, not a .conf file — the release image contains only
// the binary, a relative file path would not exist in the container.
const rbacModelText = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "ANY")
`

// The `|| p.act == "ANY"` branch is NOT optional: the seed policies below use
// 'ANY' as the act wildcard. With a plain `r.act == p.act` matcher an 'ANY'
// policy never matches any real method — every request is denied for every role.

func InitEnforcer() {
    adapter, err := mysql.NewCasbinAdapter() // built inside the DAL — see below
    if err != nil { log.Fatalf("casbin adapter: %v", err) }
    m, err := casbinmodel.NewModelFromString(rbacModelText)
    if err != nil { log.Fatalf("casbin model: %v", err) }
    e, err := casbin.NewEnforcer(m, adapter)
    if err != nil { log.Fatalf("casbin: %v", err) }
    enforcer = e
    subscribePolicyReload()
}

func subscribePolicyReload() {
    // SubscribeChannel/Publish exist on the full-profile IRedis. Casbin is a
    // full-profile escalation, so the lightweight LocalRedis not having them
    // is fine. SubscribeChannel owns the reconnect loop and error reporting —
    // no bare subscribe loop in business code (contract: backend-skeleton.md).
    go container.Default.RedisService.SubscribeChannel(context.Background(), "casbin_notify",
        func(_ string) {
            if err := enforcer.LoadPolicy(); err != nil {
                log.Printf("casbin reload: %v", err)
            }
        },
        func(err error) {
            log.Printf("casbin subscribe: %v", err) // alert hook: route to your monitoring here
        })
}

func NotifyPolicyReload(ctx context.Context) {
    container.Default.RedisService.Publish(ctx, "casbin_notify", "")
}

func CheckRoleAny(ctx context.Context, c *app.RequestContext, roles ...string) bool {
    have := GetJwtRole(ctx, c)
    for _, r := range roles {
        if r == have { return true }
    }
    return false
}

// ModeratorOrAdminMw / AdminRoleMw are NOT defined here. They live in
// biz/mw/role.go as sugar over RequireRoleMw (see lite-profile.md), so the
// default scaffold gates /papi/* with no Casbin dependency. Both names keep
// working after this escalation — replace their call sites with an
// enforcer-based middleware only when policies become runtime-editable.
```

The gorm adapter is constructed inside the DAL, not in middleware — raw `*gorm.DB` never crosses the DAL boundary (rule: `backend-skeleton.md` "biz/dal/mysql", `data-conventions.md`):

```go
// biz/dal/mysql/adapter_casbin.go — add this file only with the Casbin escalation.
package mysql

import gormadapter "github.com/casbin/gorm-adapter/v3"

func NewCasbinAdapter() (*gormadapter.Adapter, error) {
    return gormadapter.NewAdapterByDB(db)
}
```

Any other tool that genuinely needs a raw `*gorm.DB` (migration runner, health probe) gets the same treatment: a small constructor inside `biz/dal/mysql/`.

## Seed policy

Initial seed policy (insert into `casbin_rule` table once, e.g. via a one-shot `cmd/seed`):
```sql
INSERT INTO casbin_rule (ptype, v0, v1, v2) VALUES
  ('p', 'ADMIN',     '/papi/*',         'ANY'),
  ('p', 'MODERATOR', '/papi/audit/*',   'GET'),
  ('p', 'USER',      '/api/*',          'ANY');
```

But for route gating most projects do not even need to call `enforcer.Enforce(...)` per request — `ModeratorOrAdminMw` + `AdminRoleMw` (from `biz/mw/role.go`, available in the default scaffold) cover the common case. Reach for the enforcer only when policies become user-editable (e.g. team-scoped admin tools).

## biz/router/papi/middleware.go - rootMw()

```go
func rootMw() []app.HandlerFunc {
    return []app.HandlerFunc{
        mw.JwtParseMw(),
        mw.RateLimitMw(),
        mw.JwtAuthenticationMw(),
        mw.ModeratorOrAdminMw(),
    }
}
```

This runs for EVERY `/papi/*` route. Do not duplicate these in per-method `_xxxMw()` lists. (If the project adds audit middleware, a per-route `AuditMw()` is chained here too — optional, see the audit note in `backend-skeleton.md`.)

## biz/router/api/middleware.go - rootMw()

```go
func rootMw() []app.HandlerFunc {
    return []app.HandlerFunc{
        mw.JwtParseMw(),
        mw.RateLimitMw(),
    }
}
```

Per-method `_xxxMw()`:
- Anonymous endpoints (login, register, list-public): `return nil`.
- Authenticated endpoints (anything reading `GetJwtId`): `return []app.HandlerFunc{mw.JwtAuthenticationMw()}`.

## Authorization Rules Beyond Middleware

Adopt on demand (projects with multiple entry points or audit requirements). Route middleware answers "who may call this endpoint"; these service-layer rules answer "what may this caller do".

- **Every route carries an explicit role decision, enforced by a test** that walks the registered route table in BOTH directions: every route has a policy entry, and every policy entry names a registered route (plus the infra whitelist: `/ping`, `/livez`, `/healthz`). Keep a central `method + path + role` registry and explicitly register public routes too. Unknown routes or policies deny by default; do not fall back to ADMIN or another implicit allow. The test also checks that IDL declarations and registered Hertz routes match in both directions. A route without a policy entry fails CI, not ships open.
- **One validator, all entry points.** Business-rule validation lives in ONE shared place in the service layer; the public API, admin API, agent/tool APIs, and internal jobs all call it. Per-entry-point validators drift, and the drift is always a bypass.
- **Destructive operations are state-scoped and audit-preserving.** Restrict delete/update to the states where they are legal, enforced by a CAS with the state in `Where`; a wrong state is a 409, never a silent no-op. Records carrying an audit trail are never physically deleted once decided.
- **Scoped credentials filter, not just gate.** Single-resource operations CHECK the scope; list endpoints filter BY it. "Not in scope" is indistinguishable from "does not exist".
- **Fail closed.** Any error while evaluating an authorization or guardrail check denies the operation — the same rule `isJwtUserActive` already follows.

## Login Handler Pattern

```go
func UserLogin(ctx context.Context, c *app.RequestContext) {
    var req user.UserLoginReq
    if err := c.BindAndValidate(&req); err != nil {
        c.JSON(400, user.UserLoginResp{Code: consts.ErrInvalid, Msg: utils.Ptr(err.Error())})
        return
    }

    u, err := container.Default.UserService.Login(ctx, req.Identifier, req.Password)
    if err != nil {
        c.JSON(401, user.UserLoginResp{Code: consts.ErrUnauthorized, Msg: utils.Ptr(i18n.MustGetMessage(ctx, "Error.Unauthorized"))})
        return
    }

    access, err := mw.MintToken(u.ID, u.Role, consts.JwtTypeAccess, mw.AccessTTL())
    if err != nil {
        c.JSON(500, user.UserLoginResp{Code: consts.ErrInternal, Msg: utils.Ptr(i18n.MustGetMessage(ctx, "Error.Internal"))})
        return
    }
    refresh, err := mw.MintToken(u.ID, u.Role, consts.JwtTypeRefresh, mw.RefreshTTL())
    if err != nil {
        c.JSON(500, user.UserLoginResp{Code: consts.ErrInternal, Msg: utils.Ptr(i18n.MustGetMessage(ctx, "Error.Internal"))})
        return
    }
    mw.SetAuthCookies(c, access, refresh)

    c.JSON(200, user.UserLoginResp{Code: 0, User: u.ToThrift()})
}
```

## Refresh Handler Pattern

```go
func UserRefresh(ctx context.Context, c *app.RequestContext) {
    access, refresh, err := mw.RefreshJwt(ctx, c)
    if err != nil {
        c.JSON(401, base.EmptyResp{Code: consts.ErrUnauthorized, Msg: utils.Ptr(i18n.MustGetMessage(ctx, "Error.Unauthorized"))})
        return
    }
    mw.SetAuthCookies(c, access, refresh)
    c.JSON(200, base.EmptyResp{Code: 0})
}
```

Refresh and logout return `base.EmptyResp` — they carry no domain payload beyond `code`/`msg`, so they do not get their own `<Method>Resp` struct. Login keeps `user.UserLoginResp` because it returns the user object.

`/api/user/refresh` should be in IDL like any other endpoint. Its `_userRefreshMw()` returns `nil` — the refresh handler reads the refresh cookie directly, it does NOT need `JwtAuthenticationMw` (which would refuse because access is expired).

## Logout Handler Pattern

```go
func UserLogout(ctx context.Context, c *app.RequestContext) {
    mw.ClearAuthCookies(c)
    c.JSON(200, base.EmptyResp{Code: 0})
}
```

`_userLogoutMw()` returns `nil`. Logout is always available, regardless of token state.

Logout is a GET by deliberate trade-off: `SameSite=Lax` still sends cookies on cross-site top-level GET navigations, so forced logout is possible but low-severity (the user just logs back in). If unacceptable, switch the route to POST in the IDL (Lax blocks cross-site POSTs) and `userLogout()` in `frontend/skeleton.md` (`api.ts`) to `api.post`.

## Pitfalls

| Symptom | Cause | Fix |
|---------|-------|-----|
| Handler returns 200 + `{code: ErrUnauthorized}` instead of 403 when access expired | `_handlerMw()` returned `nil` for a route that reads `GetJwtId`. Root `JwtParseMw` ran and set `JWT_STATUS_REFRESH`, but `JwtAuthenticationMw` never ran. | Add `mw.JwtAuthenticationMw()` to that route's per-method mw. |
| Frontend gets stuck in a refresh loop | The refresh response did not set new cookies, OR `withCredentials: false` on axios, OR `SameSite=Strict` blocking cross-site cookie. | Check `SetAuthCookies` actually wrote both cookies; confirm `withCredentials: true`; use `SameSite=Lax`. |
| User logged out but still authed on next request | Cookies cleared with wrong `Path` or `Domain`. The browser kept the old cookie. | Match the `Path` and `Domain` used in `SetAuthCookies` exactly in `ClearAuthCookies`. |
| Casbin policy change visible on one replica but not others | `NotifyPolicyReload` not called after policy mutation, OR Redis pub/sub channel name mismatch. | Always call `mw.NotifyPolicyReload(ctx)` in the mutation handler. |
| Login response leaks token in body | Developer logged the response object. | JWT lives in HttpOnly cookies ONLY. Response body has `{code, user}`, never tokens. |
