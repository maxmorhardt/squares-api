package util

import (
	"context"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// in-memory cache with per-entry expiry
type TTLCache[K comparable, V any] struct {
	cache *expirable.LRU[K, V]
}

func NewTTLCache[K comparable, V any](size int, ttl time.Duration) *TTLCache[K, V] {
	return &TTLCache[K, V]{cache: expirable.NewLRU[K, V](size, nil, ttl)}
}

func (c *TTLCache[K, V]) GetOrLoad(ctx context.Context, key K, load func(context.Context) (V, error)) (V, error) {
	if v, ok := c.cache.Get(key); ok {
		return v, nil
	}

	v, err := load(ctx)
	if err != nil {
		var zero V
		return zero, err
	}

	c.cache.Add(key, v)
	return v, nil
}
