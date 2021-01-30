package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestCache_Sanity(t *testing.T) {
	cache := New(10, 1000)

	cache.Store("foo", "bar")

	result, ok := cache.Load("foo")

	if !ok {
		t.Error("Load did not return ok")
	}

	if result != "bar" {
		t.Errorf("Load returned wrong value: got %v want %v",
			result, "bar")
	}
}

func TestCache_TimeBasedEviction(t *testing.T) {
	cache := New(90, 80)

	cache.Store("foo", "bar")

	time.Sleep(100 * time.Millisecond)

	if _, ok := cache.Load("foo"); ok {
		t.Errorf("Load shouldn't have returned ok for foo")
	}
}

func TestCache_SizeBasedEviction(t *testing.T) {
	cache := New(10000, 80)

	keys := make([]string, 10)
	for i := range keys {
		keys[i] = fmt.Sprintf("%010d", i) // 10 bytes
		cache.Store(keys[i], keys[i])
	}

	for _, key := range keys[:2] {
		if _, ok := cache.Load(key); ok {
			t.Errorf("Load shouldn't have returned ok for %v",
				key)
		}
	}

	for _, key := range keys[2:] {
		result, ok := cache.Load(key)

		if !ok {
			t.Error("Load did not return ok")
		}

		if result != key {
			t.Errorf("Load returned wrong value: got %v want %v",
				result, key)
		}
	}
}