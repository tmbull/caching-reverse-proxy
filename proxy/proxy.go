package proxy

import (
	"bytes"
	"fmt"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/tmbull/caching-reverse-proxy/cache"
	"io"
	"net/http"
	"net/http/httputil"
)

type Proxy struct {
	Router       *httprouter.Router
	ReverseProxy *httputil.ReverseProxy
	Cache *cache.Cache
}

func (proxy *Proxy) PassThroughHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Infof("PassThroughHandler: Handling %v at URL %v.", r.Method, r.URL)
		proxy.ReverseProxy.ServeHTTP(w, r)
	}
}

func (proxy *Proxy) CachingHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Infof("CachingHandler: Handling %v at URL %v.", r.Method, r.URL)
		key := getCacheKey(r)
		if resp, ok := proxy.Cache.Load(key); ok {
			log.Debugf("CachingHandler: Found key %v in cache.", key)
			w.WriteHeader(200)
			_, err := w.Write([]byte(resp))
			if err != nil {
				log.Error(err)
			}
		} else {
			log.Debugf("CachingHandler: Did not find key %v in cache. Proxying request.", key)
			var buffer bytes.Buffer
			multi := newResponseLogger(&buffer, w)
			proxy.ReverseProxy.ServeHTTP(multi, r)
			if multi.StatusCode > 199 && multi.StatusCode < 300  {
				proxy.Cache.Store(key, string(buffer.Bytes()))
			}
		}
	}
}

func (proxy *Proxy) RegisterRoute(route Route, handler func(http.ResponseWriter, *http.Request)) {
	for _, method := range route.Methods {
		proxy.Router.HandlerFunc(method, route.Pattern, handler)
	}
}

func getCacheKey(r *http.Request) string {
	return fmt.Sprintf("%s-%s", r.Method, r.URL.String())
}

type responseLogger struct {
	resp  http.ResponseWriter
	multi io.Writer
	StatusCode int
}

func (w *responseLogger) Header() http.Header {
	return w.resp.Header()
}

func (w *responseLogger) Write(b []byte) (int, error) {
	return w.multi.Write(b)
}

func (w *responseLogger) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.resp.WriteHeader(statusCode)
}

func newResponseLogger(log io.Writer, resp http.ResponseWriter) *responseLogger {
	multi := io.MultiWriter(log, resp)
	return &responseLogger{
		resp:  resp,
		multi: multi,
	}
}