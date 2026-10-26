package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// In-process Cache for single-binary deployments. Same semantics as the Redis
// version: values round-trip through JSON, TTLs expire, unlock only releases
// the holder's own lock. State does not survive a restart and is not shared
// between processes.
type localCache struct {
	mu    sync.Mutex
	items map[string]entry
}

type entry struct {
	val []byte
	exp time.Time
	gen uint64
}

var lockGen uint64

// New is the Wire provider.
func New() (Cache, error) {
	c := &localCache{items: map[string]entry{}}
	go c.sweep()
	return c, nil
}

func (c *localCache) live(key string) (entry, bool) {
	e, ok := c.items[key]
	if ok && !e.exp.IsZero() && time.Now().After(e.exp) {
		delete(c.items, key)
		return entry{}, false
	}
	return e, ok
}

func (c *localCache) Get(_ context.Context, key string, dst any) error {
	c.mu.Lock()
	e, ok := c.live(key)
	c.mu.Unlock()
	if !ok {
		return ErrMiss
	}
	return json.Unmarshal(e.val, dst)
}

func (c *localCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = entry{val: b, exp: expiry(ttl)}
	return nil
}

func (c *localCache) Delete(_ context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		delete(c.items, k)
	}
	return nil
}

func (c *localCache) TryLock(_ context.Context, key string, ttl time.Duration) (func(), bool, error) {
	key = "lock:" + key
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, held := c.live(key); held {
		return nil, false, nil
	}
	lockGen++
	gen := lockGen
	c.items[key] = entry{exp: expiry(ttl), gen: gen}
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if e, ok := c.items[key]; ok && e.gen == gen {
			delete(c.items, key)
		}
	}, true, nil
}

func (c *localCache) Ping(context.Context) error { return nil }

func (c *localCache) sweep() {
	for range time.Tick(time.Minute) {
		c.mu.Lock()
		for k := range c.items {
			c.live(k)
		}
		c.mu.Unlock()
	}
}

func expiry(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return time.Now().Add(ttl)
}
