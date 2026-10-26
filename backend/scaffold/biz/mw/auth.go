package mw

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/errs"
	"MODULE_NAME/biz/handler/resp"
	"MODULE_NAME/config"
	"MODULE_NAME/container"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/golang-jwt/jwt/v5"
)

const (
	accessCookie        = "ACCESS_TOKEN"
	refreshCookie       = "REFRESH_TOKEN"
	HeaderRefreshNeeded = "X-Token-Refresh-Needed"
	ctxUser             = "user"
)

// publicRoutes are reachable without a session ("METHOD /path" as registered).
// Every other /api and /papi route requires one; that is the point of putting
// Auth in rootMw instead of on individual routes.
var publicRoutes = map[string]bool{
	"POST /api/auth/login":   true,
	"POST /api/auth/refresh": true,
	"POST /api/auth/logout":  true,
}

// Auth runs first on every /api and /papi route. A valid access token loads the
// user (from the database, so disabling a user or changing a role applies
// immediately). An expired access token with a valid refresh token answers 401
// with X-Token-Refresh-Needed so the client refreshes and retries.
func Auth() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		uid, accessErr := verify(c.Cookie(accessCookie), "access")
		if accessErr == nil {
			u, err := container.Default.User.Active(ctx, uid)
			if err == nil {
				c.Set(ctxUser, u)
			} else if !isClientError(err) {
				resp.Fail(ctx, c, err)
				c.Abort()
				return
			}
		}
		if publicRoutes[string(c.Method())+" "+c.FullPath()] || CurrentUser(c) != nil {
			c.Next(ctx)
			return
		}
		if _, err := verify(c.Cookie(refreshCookie), "refresh"); err == nil {
			c.Header(HeaderRefreshNeeded, "true")
			resp.Abort(c, errs.Unauthorized("登录已过期"))
			return
		}
		resp.Abort(c, errs.Unauthorized("请先登录"))
	}
}

// RequireRole goes after Auth. Roles rank USER < MODERATOR < ADMIN.
func RequireRole(min string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if u := CurrentUser(c); u == nil || rank[u.Role] < rank[min] {
			resp.Abort(c, errs.Forbidden("无权访问"))
			return
		}
		c.Next(ctx)
	}
}

var rank = map[string]int{model.RoleUser: 1, model.RoleModerator: 2, model.RoleAdmin: 3}

// CurrentUser is the logged-in user, or nil on a public route without a session.
func CurrentUser(c *app.RequestContext) *model.User {
	u, _ := c.Get(ctxUser)
	user, _ := u.(*model.User)
	return user
}

// IssueSession sets both cookies. Called on login and on refresh; re-sending the
// refresh cookie slides the 30-day window for active users.
func IssueSession(c *app.RequestContext, uid int64) error {
	j := config.C.JWT
	access, err := sign(uid, "access", time.Duration(j.AccessTTLSeconds)*time.Second)
	if err != nil {
		return err
	}
	refresh, err := sign(uid, "refresh", time.Duration(j.RefreshTTLSeconds)*time.Second)
	if err != nil {
		return err
	}
	setCookie(c, accessCookie, access, j.AccessTTLSeconds)
	setCookie(c, refreshCookie, refresh, j.RefreshTTLSeconds)
	return nil
}

// RefreshSession validates the refresh cookie and issues a new session.
func RefreshSession(ctx context.Context, c *app.RequestContext) error {
	uid, err := verify(c.Cookie(refreshCookie), "refresh")
	if err != nil {
		return errs.Unauthorized("登录已过期，请重新登录")
	}
	if _, err := container.Default.User.Active(ctx, uid); err != nil {
		return err
	}
	return IssueSession(c, uid)
}

func ClearSession(c *app.RequestContext) {
	setCookie(c, accessCookie, "", -1)
	setCookie(c, refreshCookie, "", -1)
}

func setCookie(c *app.RequestContext, name, value string, maxAge int) {
	c.SetCookie(name, value, maxAge, "/", "", protocol.CookieSameSiteLaxMode, config.C.JWT.CookieSecure, true)
}

type claims struct {
	Type string `json:"typ"`
	jwt.RegisteredClaims
}

func sign(uid int64, typ string, ttl time.Duration) (string, error) {
	now := time.Now()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Type: typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(uid, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}).SignedString([]byte(config.C.JWT.Secret))
}

func verify(raw []byte, typ string) (int64, error) {
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return 0, errors.New("no token")
	}
	var cl claims
	_, err := jwt.ParseWithClaims(token, &cl, func(*jwt.Token) (any, error) {
		return []byte(config.C.JWT.Secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return 0, err
	}
	if cl.Type != typ {
		return 0, errors.New("wrong token type")
	}
	return strconv.ParseInt(cl.Subject, 10, 64)
}

func isClientError(err error) bool {
	var e *errs.Error
	return errors.As(err, &e)
}
