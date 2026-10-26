package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"MODULE_NAME/biz/errs"
	"MODULE_NAME/biz/handler/resp"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
)

//go:embed all:frontend/dist
var dist embed.FS

func registerFrontend(r *server.Hertz) {
	sub, err := fs.Sub(dist, "frontend/dist")
	if err != nil {
		panic(err)
	}
	files := adaptor.HertzHandler(http.FileServer(http.FS(sub)))
	// Vite content-hashes everything under assets/, so it can be cached forever.
	r.GET("/assets/*path", func(ctx context.Context, c *app.RequestContext) {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		files(ctx, c)
	})
	r.GET("/fonts/*path", files)

	index, _ := fs.ReadFile(sub, "index.html")
	r.NoRoute(func(ctx context.Context, c *app.RequestContext) {
		p := string(c.Path())
		if p == "/api" || p == "/papi" || strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/papi/") {
			resp.Fail(ctx, c, errs.NotFound("接口不存在"))
			return
		}
		if index == nil {
			c.String(http.StatusNotFound, "frontend not built: run ./build.sh")
			return
		}
		// index.html names the current hashed assets, so it must never be cached.
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})
}
