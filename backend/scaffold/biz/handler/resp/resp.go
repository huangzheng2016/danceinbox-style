// Package resp writes the error envelope {code, msg} shared by every endpoint.
// Success responses are the hz-generated Resp structs, whose code defaults to 0.
package resp

import (
	"context"
	"errors"
	"net/http"

	"MODULE_NAME/biz/errs"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/utils"
)

// Bind binds and validates the request. On failure it writes a 400 and returns false.
func Bind(c *app.RequestContext, req any) bool {
	if err := c.BindAndValidate(req); err != nil {
		write(c, errs.Invalid("请求参数错误:"+err.Error()))
		return false
	}
	return true
}

// Fail writes err. An *errs.Error is shown to the client as-is; any other error
// is logged with the request id and reported as a generic 500.
func Fail(ctx context.Context, c *app.RequestContext, err error) {
	var e *errs.Error
	if errors.As(err, &e) {
		write(c, e)
		return
	}
	hlog.CtxErrorf(ctx, "request_id=%s %s %s: %v", c.GetString("request_id"), c.Method(), c.Path(), err)
	write(c, &errs.Error{Status: http.StatusInternalServerError, Code: errs.CodeInternal, Msg: "服务器内部错误"})
}

// Abort is Fail for middleware: it also stops the handler chain.
func Abort(c *app.RequestContext, e *errs.Error) {
	c.AbortWithStatusJSON(e.Status, utils.H{"code": e.Code, "msg": e.Msg})
}

func write(c *app.RequestContext, e *errs.Error) {
	c.JSON(e.Status, utils.H{"code": e.Code, "msg": e.Msg})
}
