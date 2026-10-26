package mw

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/google/uuid"
)

// AccessLog tags each request with an X-Request-Id (reusing the caller's) and logs
// one line per request. resp.Fail logs the same id for 500s.
func AccessLog() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		id := string(c.GetHeader("X-Request-Id"))
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header("X-Request-Id", id)
		c.Next(ctx)
		hlog.CtxInfof(ctx, "%s %s %d %s request_id=%s", c.Method(), c.Path(), c.Response.StatusCode(), time.Since(start).Round(time.Millisecond), id)
	}
}
