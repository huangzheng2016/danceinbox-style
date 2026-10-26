package handler

import (
	"context"
	"net/http"
	"time"

	"MODULE_NAME/biz/dal/db"
	"MODULE_NAME/container"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
)

var Version = "dev"

// Livez answers as long as the process runs. It checks no dependency, so a
// database outage does not get the container restarted in a loop.
func Livez(ctx context.Context, c *app.RequestContext) {
	c.JSON(http.StatusOK, utils.H{"status": "ok"})
}

// Healthz is readiness: 200 only when every dependency answers.
func Healthz(ctx context.Context, c *app.RequestContext) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	deps := utils.H{"db": probe(db.Ping(ctx)), "cache": probe(container.Default.Cache.Ping(ctx))}
	status, code := "ok", http.StatusOK
	for _, v := range deps {
		if v != "up" {
			status, code = "degraded", http.StatusServiceUnavailable
		}
	}
	c.JSON(code, utils.H{"status": status, "version": Version, "deps": deps})
}

func probe(err error) string {
	if err != nil {
		return "down: " + err.Error()
	}
	return "up"
}
