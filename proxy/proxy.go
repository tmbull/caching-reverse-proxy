package proxy

import (
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
)

type Proxy struct {
	Router       *httprouter.Router
	ReverseProxy *httputil.ReverseProxy
}

func (proxy *Proxy) PassThroughHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info(r.URL)
		proxy.ReverseProxy.ServeHTTP(w, r)
	}
}

func (proxy *Proxy) CachingHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info(r.URL)
		proxy.ReverseProxy.ServeHTTP(w, r)
	}
}

func (proxy *Proxy) RegisterRoute(route Route, handler func(http.ResponseWriter, *http.Request)) {
	for _, method := range route.Methods {
		proxy.Router.HandlerFunc(method, route.Pattern, handler)
	}
}