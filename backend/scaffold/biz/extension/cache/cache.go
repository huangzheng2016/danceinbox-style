// Package cache is the only way business code reaches Redis. The lite profile
// replaces provider.go with an in-process version; callers do not change.
package cache

import (
	"context"
	"errors"
	"time"
)

var ErrMiss = errors.New("cache miss")

type Cache interface {
	// Get decodes the JSON value at key into dst, or returns ErrMiss.
	Get(ctx context.Context, key string, dst any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	// TryLock takes key for ttl without waiting. ok is false when another holder
	// has it. unlock releases only this holder's lock.
	TryLock(ctx context.Context, key string, ttl time.Duration) (unlock func(), ok bool, err error)
	Ping(ctx context.Context) error
}
