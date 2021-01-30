package main

import (
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/tkanos/gonfig"
	"github.com/tmbull/caching-reverse-proxy/cache"
	. "github.com/tmbull/caching-reverse-proxy/proxy"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	config := Config{}
	if err := gonfig.GetConf("config.json", &config); err != nil {
		log.Fatal(err)
	}

	level, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)

	targetUrl, err := url.Parse(config.TargetUrl)
	if err != nil {
		log.Fatal(err)
	}

	rp := httputil.NewSingleHostReverseProxy(targetUrl)
	router := httprouter.New()
	c := cache.New(config.CacheTtlInMillis, config.CacheCapacityInBytes)
	proxy := Proxy{
		Router:       router,
		ReverseProxy: rp,
		Cache: c,
	}

	log.Info("Registering cached routes.")
	for _, route := range config.CachedRoutes {
		log.Debugf("Registering route: %v", route)
		proxy.RegisterRoute(route, proxy.CachingHandler())
	}

	log.Info("Registering pass through routes.")
	for _, route := range config.PassThroughRoutes {
		log.Debugf("Registering route: %v", route)
		proxy.RegisterRoute(route, proxy.PassThroughHandler())
	}

	err = http.ListenAndServe(config.ListenAddr, router)
	if err != nil {
		log.Fatal(err)
	}
}

