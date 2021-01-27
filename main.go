package main

import (
	"github.com/julienschmidt/httprouter"
	"github.com/tkanos/gonfig"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Route struct {
	Methods []string
	Pattern string
}

type Config struct {
	ListenAddr string
	TargetUrl string
	CachedRoutes []Route
	PassThroughRoutes []Route
}

func main() {
	config := Config{}
	err := gonfig.GetConf("config.json", &config)

	if err != nil {
		panic(err)
	}

	targetUrl, err := url.Parse(config.TargetUrl)

	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetUrl)
	router := httprouter.New()
	for _, route := range config.CachedRoutes {
		for _, method := range route.Methods {
			router.HandlerFunc(method, route.Pattern, cachingHandler(proxy))
		}
	}

	for _, route := range config.PassThroughRoutes {
		for _, method := range route.Methods {
			router.HandlerFunc(method, route.Pattern, passThroughHandler(proxy))
		}
	}

	err = http.ListenAndServe(config.ListenAddr, router)
	if err != nil {
		panic(err)
	}
}

func passThroughHandler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL)
		w.Header().Set("X-Ben", "Rad")
		p.ServeHTTP(w, r)
	}
}

func cachingHandler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL)
		w.Header().Set("X-Ben", "Rad")
		p.ServeHTTP(w, r)
	}
}