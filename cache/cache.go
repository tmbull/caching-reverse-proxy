package cache

import (
	"container/list"
	"sync"
	"time"
)

type Entry struct {
	key string
	value string
	size int
	timestamp int64
}

type Cache struct {
	ttl int64
	capacity int
	size     int
	list     *list.List
	syncMap  *sync.Map
}

func New(ttlInMillis int64, capacityInBytes int) *Cache {
	return &Cache {
		ttl: ttlInMillis,
		capacity: capacityInBytes,
		size:     0,
		list:     new(list.List),
		syncMap:  new(sync.Map),
	}
}

func (cache *Cache) Load(key string) (string, bool) {
	if entry, ok := cache.syncMap.Load(key); ok {
		cacheElement := entry.(*list.Element)
		cacheEntry := cacheElement.Value.(*Entry)

		if cacheEntry.timestamp > (nowInMillis() - cache.ttl) {
			return cacheEntry.value, true
		} else {
			cache.size -= cacheEntry.size
			cache.syncMap.Delete(cacheEntry.key)
			cache.list.Remove(cacheElement)
		}
	}

	return "", false
}

func (cache *Cache) Store(key string, value string) {
	valSize := len(value)
	if entry, ok := cache.syncMap.Load(key); ok {
		cache.list.MoveToFront(entry.(*list.Element))
		cache.size -= entry.(*list.Element).Value.(*Entry).size
		cache.size += valSize

		entry.(*list.Element).Value = &Entry{
			key:   key,
			value: value,
			size:  valSize,
			timestamp: nowInMillis(),
		}
	} else {
		for cache.size + valSize > cache.capacity {
			oldest := cache.list.Back().Value.(*Entry)

			cache.size -= oldest.size
			cache.syncMap.Delete(oldest.key)
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
		cache.syncMap.Store(key, p)
	}
}

func nowInMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}