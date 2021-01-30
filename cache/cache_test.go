package cache

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestCache_Sanity(t *testing.T) {
	cache := New(10, 1000)
	val := []byte("bar")
	cache.Store("foo", val)

	result, ok := cache.Load("foo")

	if !ok {
		t.Error("Load did not return ok")
	}

	if !bytes.Equal(result, val) {
		t.Errorf("Load returned wrong value: got %v want %v",
			result, val)
	}
}

func TestCache_TimeBasedEviction(t *testing.T) {
	cache := New(90, 80)

	cache.Store("foo", []byte("bar"))

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
		cache.Store(keys[i], []byte(keys[i]))
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

		if !bytes.Equal(result, []byte(key)) {
			t.Errorf("Load returned wrong value: got %v want %v",
				result, key)
		}
	}
}