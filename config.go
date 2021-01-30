package main

import . "github.com/tmbull/caching-reverse-proxy/proxy"

type Config struct {
	LogLevel string
	ListenAddr string
	TargetUrl string
	CacheTtlInMillis int64
	CacheCapacityInBytes int
	CachedRoutes []Route
	PassThroughRoutes []Route
}
