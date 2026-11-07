// Package actor carries the signed-in user's id in the request context, so
// services can attribute changes (audit, created_by) without handlers passing
// it through every call. mw.Auth sets it; services read it.
package actor

import "context"

type key struct{}

func With(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, key{}, userID)
}

// ID is the current user's id, or 0 outside a signed-in request.
func ID(ctx context.Context) int64 {
	id, _ := ctx.Value(key{}).(int64)
	return id
}
