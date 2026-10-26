package mw

import (
	"context"
	"sync"
	"time"

	"MODULE_NAME/biz/errs"
	"MODULE_NAME/biz/handler/resp"

	"github.com/cloudwego/hertz/pkg/app"
	"golang.org/x/time/rate"
)

// RateLimit is a per-client-IP token bucket kept in process memory. It is correct
// for a single replica; with several replicas, move the buckets into the cache.
//
//	rootMw:     mw.RateLimit(20, 40)                          general traffic
//	_loginMw:   mw.RateLimit(rate.Every(10*time.Second), 5)   password guessing
func RateLimit(r rate.Limit, burst int) app.HandlerFunc {
	type bucket struct {
		lim  *rate.Limiter
		seen time.Time
	}
	var mu sync.Mutex
	buckets := map[string]*bucket{}

	go func() {
		for range time.Tick(10 * time.Minute) {
			mu.Lock()
			for ip, b := range buckets {
				if time.Since(b.seen) > 10*time.Minute {
					delete(buckets, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(ctx context.Context, c *app.RequestContext) {
		ip := c.ClientIP()
		mu.Lock()
		b, ok := buckets[ip]
		if !ok {
			b = &bucket{lim: rate.NewLimiter(r, burst)}
			buckets[ip] = b
		}
		b.seen = time.Now()
		allowed := b.lim.Allow()
		mu.Unlock()
		if !allowed {
			resp.Abort(c, &errs.Error{Status: 429, Code: errs.CodeInvalid, Msg: "请求过于频繁，请稍后再试"})
			return
		}
		c.Next(ctx)
	}
}
