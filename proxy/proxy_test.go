package proxy

import (
	"bytes"
	"fmt"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/tmbull/caching-reverse-proxy/cache"
	"math"
	"math/rand"
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
		if user, pass, ok := r.BasicAuth(); ok && user == "user" && pass == "secretPassword" {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(strconv.Itoa(counter)))
			counter++
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
}

func getRouter(
	u *url.URL,
	ttlInMillis int64,
	capacityInBytes int,
	handler func(*Proxy) func(http.ResponseWriter, *http.Request),
) *httprouter.Router {
	rp := httputil.NewSingleHostReverseProxy(u)
	router := httprouter.New()
	c := cache.New(ttlInMillis, capacityInBytes)
	proxy := Proxy{
		Router:       router,
		ReverseProxy: rp,
		Cache: c,
	}

	routes := []Route{
		{
		Methods: []string{"GET"},
		Pattern: "/api/things/:id",
		},
		{
			Methods: []string{"GET"},
			Pattern: "/api/things",
		},
	}

	for _, route := range routes {
		proxy.RegisterRoute(route, handler(&proxy))
	}

	return router
}

func TestProxy_PassThroughHandler(t *testing.T) {
	backend := getBackend()
	u, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	router := getRouter(u, 1000, 1024*1024, (*Proxy).PassThroughHandler)

	authTests(t, router)

	routingTests(t, router)

	t.Run("It doesn't cache", func(t *testing.T) {
		req := getRequest(t, "user", "secretPassword")

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

		if bytes.Equal(count0.Bytes(), count1.Bytes()) {
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

	router := getRouter(u, 1000, 1024*1024, (*Proxy).CachingHandler)

	authTests(t, router)

	routingTests(t, router)

	t.Run("It caches", func(t *testing.T) {
		req := getRequest(t, "user", "secretPassword")

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

		if !bytes.Equal(count0.Bytes(), count1.Bytes()) {
			t.Errorf("expected same counts: got %v and %v",
				count0, count1)
		}
	})
}

func getRequest(t *testing.T, username string, password string) *http.Request {
	req, err := http.NewRequest("GET", "/api/things", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth(username, password)

	return req
}

func authTests(t *testing.T, router *httprouter.Router) {
	t.Run("It passes through correct Authorization header", func(t *testing.T) {
		req := getRequest(t, "user", "secretPassword")


		rr1 := httptest.NewRecorder()
		router.ServeHTTP(rr1, req)
		if status := rr1.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	})

	t.Run("It passes through incorrect Authorization header", func(t *testing.T) {
		req := getRequest(t, "user", "wrongPassword")


		rr1 := httptest.NewRecorder()
		router.ServeHTTP(rr1, req)
		if status := rr1.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusUnauthorized)
		}
	})
}

func routingTests(t *testing.T, router *httprouter.Router) {
	t.Run("Matching route and method", func(t *testing.T) {
		req := getRequest(t, "user", "secretPassword")

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
}

func getBenchMarkingBackend() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			numBytes, err := strconv.Atoi(r.URL.Query().Get("bytes"))
			if err != nil {
				numBytes = 10
			}
			token := make([]byte, numBytes)
			rand.Read(token)
	}))
}

func BenchmarkProxy_CachingHandler(b *testing.B) {
	log.SetLevel(log.WarnLevel)
	/*
	Backend: takes # bytes as query param, returns # random bytes

	Cache Benchmarking params:
		# Cache capacity
		# Cache size / capacity
		# Hit rate
		# Concurrency: # of concurrent clients

	 */
	cacheCapacity := 1024*1024*1024


	backend := getBenchMarkingBackend()
	u, err := url.Parse(backend.URL)
	if err != nil {
		b.Fatal(err)
	}

	router := getRouter(u, 15*60*1000, cacheCapacity, (*Proxy).CachingHandler)

	requestSizes := []int{100, 1000, 1000000}

	cacheFullness := []float64{0.2, 0.5, 1.0}

	hitRates := []float64{0.0, 0.5, 1.0}

	//numClients := 1

	for _, fullness := range cacheFullness {
		for _, reqSize := range requestSizes {
			// This will introduce a slight error to later tests since the cache is not completely rebuilt, but it
			// drastically reduces the time to run all of the tests with a large cache
			initialSize := int(float64(cacheCapacity) / float64(reqSize) * fullness)
			for i := 0; i < initialSize; i++ {
				req, err := http.NewRequest("GET", fmt.Sprintf("/api/things/%d?bytes=%d", i, reqSize), nil)
				if err != nil {
					b.Fatal(err)
				}
				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)

				if status := rr.Code; status != http.StatusOK {
					b.Fatalf("handler returned wrong status code: got %v want %v",
						status, http.StatusOK)
				}
			}
			for _, hitRate := range hitRates {
				//for numClients := range numClients {
				//
				//}
				b.Run(fmt.Sprintf("Request Size %d Cache Fullness %f Hit Rate %f",
					reqSize, fullness, hitRate), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						b.StopTimer()
						var max int
						if hitRate == 0.0 || initialSize == 0 {
							max = math.MaxInt32
						} else {
							max = int(float64(initialSize) * (1 / hitRate))
						}

						id := rand.Intn(max)
						req, err := http.NewRequest("GET", fmt.Sprintf("/api/things/%d?bytes=%d", id, reqSize), nil)
						if err != nil {
							b.Fatal(err)
						}
						rr := httptest.NewRecorder()

						b.StartTimer()
						router.ServeHTTP(rr, req)

						if status := rr.Code; status != http.StatusOK {
							b.Fatalf("handler returned wrong status code: got %v want %v",
								status, http.StatusOK)
						}
					}
				})
			}
		}
	}
}