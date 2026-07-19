package util

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTTLCache_CachesWithinWindow(t *testing.T) {
	var calls int32
	cache := NewTTLCache[string, int](16, time.Minute)
	load := func(context.Context) (int, error) {
		atomic.AddInt32(&calls, 1)
		return 42, nil
	}

	first, err := cache.GetOrLoad(context.Background(), "k", load)
	require.NoError(t, err)
	second, err := cache.GetOrLoad(context.Background(), "k", load)
	require.NoError(t, err)

	assert.Equal(t, 42, first)
	assert.Equal(t, 42, second)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestTTLCache_SeparatesKeys(t *testing.T) {
	cache := NewTTLCache[string, string](16, time.Minute)
	load := func(v string) func(context.Context) (string, error) {
		return func(context.Context) (string, error) { return v, nil }
	}

	a, err := cache.GetOrLoad(context.Background(), "a", load("first"))
	require.NoError(t, err)
	b, err := cache.GetOrLoad(context.Background(), "b", load("second"))
	require.NoError(t, err)

	assert.Equal(t, "first", a)
	assert.Equal(t, "second", b)
}

func TestTTLCache_ReloadsAfterExpiry(t *testing.T) {
	var calls int32
	cache := NewTTLCache[string, int](16, 5*time.Millisecond)
	load := func(context.Context) (int, error) {
		return int(atomic.AddInt32(&calls, 1)), nil
	}

	first, err := cache.GetOrLoad(context.Background(), "k", load)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	second, err := cache.GetOrLoad(context.Background(), "k", load)
	require.NoError(t, err)

	assert.Equal(t, 1, first)
	assert.Equal(t, 2, second)
}

func TestTTLCache_DoesNotCacheErrors(t *testing.T) {
	var calls int32
	boom := errors.New("boom")
	cache := NewTTLCache[string, int](16, time.Minute)
	load := func(context.Context) (int, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			return 0, boom
		}
		return 7, nil
	}

	_, err := cache.GetOrLoad(context.Background(), "k", load)
	require.ErrorIs(t, err, boom)

	value, err := cache.GetOrLoad(context.Background(), "k", load)
	require.NoError(t, err)
	assert.Equal(t, 7, value)
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}

func TestTTLCache_ConcurrentAccessIsSafe(t *testing.T) {
	cache := NewTTLCache[string, int](16, time.Minute)
	load := func(ctx context.Context) (int, error) {
		return 1, ctx.Err()
	}

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := cache.GetOrLoad(context.Background(), "k", load)
			assert.NoError(t, err)
			assert.Equal(t, 1, v)
		}()
	}
	wg.Wait()
}
