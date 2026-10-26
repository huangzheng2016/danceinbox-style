package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"MODULE_NAME/config"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type redisCache struct{ c *redis.Client }

// New is the Wire provider.
func New() (Cache, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     config.C.Redis.Addr,
		Password: config.C.Redis.Password,
		DB:       config.C.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &redisCache{c: c}, nil
}

func (r *redisCache) Get(ctx context.Context, key string, dst any) error {
	b, err := r.c.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrMiss
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

func (r *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.c.Set(ctx, key, b, ttl).Err()
}

func (r *redisCache) Delete(ctx context.Context, keys ...string) error {
	return r.c.Del(ctx, keys...).Err()
}

var unlockScript = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) end return 0`)

func (r *redisCache) TryLock(ctx context.Context, key string, ttl time.Duration) (func(), bool, error) {
	key = "lock:" + key
	token := uuid.NewString()
	ok, err := r.c.SetNX(ctx, key, token, ttl).Result()
	if err != nil || !ok {
		return nil, false, err
	}
	return func() { unlockScript.Run(context.WithoutCancel(ctx), r.c, []string{key}, token) }, true, nil
}

func (r *redisCache) Ping(ctx context.Context) error {
	return r.c.Ping(ctx).Err()
}
