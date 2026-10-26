package mw

import (
	"context"
	"net/http"
	"testing"
	"time"

	"MODULE_NAME/biz/dal/db/dbtest"
	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/service/user"
	"MODULE_NAME/config"
	"MODULE_NAME/container"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
)

func setup(t *testing.T) (*route.Engine, map[string]string) {
	config.C.JWT.Secret = "test-secret-test-secret-test-secret"
	config.C.JWT.AccessTTLSeconds = 60
	config.C.JWT.RefreshTTLSeconds = 600
	users := user.NewService(dbtest.New(t))
	container.Default = &container.Container{User: users}

	ctx := context.Background()
	admin, _ := users.Create(ctx, "admin", "password1", model.RoleAdmin)
	plain, _ := users.Create(ctx, "bob", "password1", model.RoleUser)
	access := func(id int64) string { tok, _ := sign(id, "access", time.Minute); return accessCookie + "=" + tok }
	refresh, _ := sign(plain.ID, "refresh", time.Hour)

	e := server.New().Engine
	ok := func(ctx context.Context, c *app.RequestContext) { c.String(http.StatusOK, "ok") }
	e.POST("/api/auth/login", Auth(), ok)
	e.GET("/api/thing", Auth(), ok)
	e.GET("/papi/thing", Auth(), RequireRole(model.RoleModerator), ok)
	return e, map[string]string{
		"admin":   access(admin.ID),
		"user":    access(plain.ID),
		"refresh": refreshCookie + "=" + refresh,
		"forged":  accessCookie + "=" + forge(admin.ID),
	}
}

// forge signs a token with a key the server does not know.
func forge(id int64) string {
	saved := config.C.JWT.Secret
	defer func() { config.C.JWT.Secret = saved }()
	config.C.JWT.Secret = "attacker-secret-attacker-secret-xx"
	tok, _ := sign(id, "access", time.Minute)
	return tok
}

func TestAuth(t *testing.T) {
	e, cookie := setup(t)
	cases := []struct {
		name, method, path, cookie string
		status                     int
		refreshHeader              bool
	}{
		{"public route without session", "POST", "/api/auth/login", "", 200, false},
		{"protected route without session", "GET", "/api/thing", "", 401, false},
		{"protected route with session", "GET", "/api/thing", cookie["user"], 200, false},
		{"forged token", "GET", "/api/thing", cookie["forged"], 401, false},
		{"refresh token only asks for refresh", "GET", "/api/thing", cookie["refresh"], 401, true},
		{"refresh token is not an access token", "GET", "/api/thing", accessCookie + cookie["refresh"][len(refreshCookie):], 401, false},
		{"role too low", "GET", "/papi/thing", cookie["user"], 403, false},
		{"role sufficient", "GET", "/papi/thing", cookie["admin"], 200, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var headers []ut.Header
			if tc.cookie != "" {
				headers = append(headers, ut.Header{Key: "Cookie", Value: tc.cookie})
			}
			w := ut.PerformRequest(e, tc.method, tc.path, nil, headers...)
			r := w.Result()
			if r.StatusCode() != tc.status {
				t.Fatalf("status %d, want %d: %s", r.StatusCode(), tc.status, r.Body())
			}
			if got := string(r.Header.Peek(HeaderRefreshNeeded)) == "true"; got != tc.refreshHeader {
				t.Fatalf("refresh header %v, want %v", got, tc.refreshHeader)
			}
		})
	}
}
