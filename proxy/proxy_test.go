package proxy

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"
	"testing"
)

func getBackend() *httptest.Server {
	counter := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(strconv.Itoa(counter)))
		counter++
	}))
}

func getRouter(u *url.URL, handler func(*Proxy) func(http.ResponseWriter, *http.Request)) *httprouter.Router {
	rp := httputil.NewSingleHostReverseProxy(u)
	router := httprouter.New()
	proxy := Proxy{
		Router:       router,
		ReverseProxy: rp,
	}

	route := Route{
		Methods: []string{"GET"},
		Pattern: "/api/things",
	}
	proxy.RegisterRoute(route, handler(&proxy))

	return router
}

func TestProxy_PassThroughHandler(t *testing.T) {
	backend := getBackend()
	u, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	router := getRouter(u, (*Proxy).PassThroughHandler)

	t.Run("Matching route and method", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/things", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	})

	t.Run("Matching route with different method", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/things", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusMethodNotAllowed)
		}
	})

	t.Run("Matching method with different route", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/other", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusNotFound)
		}
	})

	t.Run("It doesn't cache", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/things", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr1 := httptest.NewRecorder()
		router.ServeHTTP(rr1, req)
		if status := rr1.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
		count0 := rr1.Body

		rr2 := httptest.NewRecorder()
		router.ServeHTTP(rr2, req)
		if status := rr1.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
		count1 := rr2.Body

		if count0 == count1 {
			t.Errorf("expected different counts, but both were %v",
				count0)
		}
	})
}

func TestProxy_CachingHandler(t *testing.T) {
	backend := getBackend()
	u, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	router := getRouter(u, (*Proxy).PassThroughHandler)

	t.Run("It caches", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/things", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr1 := httptest.NewRecorder()
		router.ServeHTTP(rr1, req)
		if status := rr1.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
		count0 := rr1.Body

		rr2 := httptest.NewRecorder()
		router.ServeHTTP(rr2, req)
		if status := rr1.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
		count1 := rr2.Body

		if count0 != count1 {
			t.Errorf("expected same counts: got %v and %v",
				count0, count1)
		}
	})
}