package cache

import (
	"container/list"
	"sync"
	"time"
)

type Entry struct {
	key string
	value []byte
	size int
	timestamp int64
}

type Cache struct {
	ttl      int64
	capacity int
	size     int
	list     *list.List
	innerMap map[string]*list.Element
	mu sync.RWMutex
}

func New(ttlInMillis int64, capacityInBytes int) *Cache {
	return &Cache {
		ttl:      ttlInMillis,
		capacity: capacityInBytes,
		size:     0,
		list:     new(list.List),
		innerMap: make(map[string]*list.Element),
	}
}

func (cache *Cache) Load(key string) ([]byte, bool) {
	cache.mu.RLock()
	entry, ok := cache.innerMap[key]
	cache.mu.RUnlock()
	if ok {
		cacheElement := entry
		cacheEntry := cacheElement.Value.(*Entry)

		if cacheEntry.timestamp > (nowInMillis() - cache.ttl) {
			return cacheEntry.value, true
		} else {
			cache.mu.Lock()
			cache.size -= cacheEntry.size
			delete(cache.innerMap, cacheEntry.key)
			cache.list.Remove(cacheElement)
			cache.mu.Unlock()
		}
	}

	return nil, false
}

func (cache *Cache) Store(key string, value []byte) {
	valSize := len(value)
	cache.mu.Lock()
	if entry, ok := cache.innerMap[key]; ok {
		cache.list.MoveToFront(entry.Value.(*list.Element))
		cache.size -= entry.Value.(*list.Element).Value.(*Entry).size
		cache.size += valSize

		entry.Value.(*list.Element).Value = &Entry{
			key:   key,
			value: value,
			size:  valSize,
			timestamp: nowInMillis(),
		}
	} else {
		for cache.size + valSize > cache.capacity {
			oldest := cache.list.Back().Value.(*Entry)

			cache.size -= oldest.size
			delete(cache.innerMap, oldest.key)
			cache.list.Remove(cache.list.Back())
		}

		entry := &Entry{
				key:   key,
				value: value,
				size:  valSize,
				timestamp: nowInMillis(),
			}
		cache.size += valSize
		p := cache.list.PushFront(entry)
		cache.innerMap[key] = p
	}
	cache.mu.Unlock()
}

func nowInMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}