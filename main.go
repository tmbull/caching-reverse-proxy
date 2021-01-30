package main

import (
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/tkanos/gonfig"
	. "github.com/tmbull/caching-reverse-proxy/proxy"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	config := Config{}
	err := gonfig.GetConf("config.json", &config)

	if err != nil {
		log.Fatal(err)
	}

	targetUrl, err := url.Parse(config.TargetUrl)

	if err != nil {
		log.Fatal(err)
	}

	rp := httputil.NewSingleHostReverseProxy(targetUrl)
	router := httprouter.New()
	proxy := Proxy{
		Router:       router,
		ReverseProxy: rp,
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

