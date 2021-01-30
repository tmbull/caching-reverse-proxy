package main

import . "github.com/tmbull/caching-reverse-proxy/proxy"

type Config struct {
	ListenAddr string
	TargetUrl string
	CachedRoutes []Route
	PassThroughRoutes []Route
}
