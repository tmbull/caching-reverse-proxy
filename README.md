# caching-reverse-proxy

This is a reverse proxy that optionally caches the responses from the backend service. The proxy is configurable: it can
have any number of "pass-through" routes which do not do any caching, and any number of "cached" routes. Routes that are
cached will have successful responses saved to the cache for a period of time (configurable via `CacheTtlInMillis` 
describe below). 

No attempt is made to update entries in the cache; they will only be replaced after they have expired from the cache or 
been evicted due to capacity. Once the cache has reached maximum capacity (configurable via `CacheCapacityInBytes`
described below), items will be evicted on insert to keep the total size of the cache below the maximum capacity.

## Main project (root directory)

This is the reverse proxy. It is configurable via a JSON file (see sample [here](config.json)). Configuration options
are as follows:

* `ListenAddr` - The `host:port` to listen on.
* `TargetUrl` - The base URL to target proxied requests to.
* `CacheTtlInMillis` - The time for a page to live in the cache before being evicted (Value in default Config is 15 minutes)
* `CacheCapacityInBytes` - The capacity of the cache in bytes (Value in default Config is 1GB)
* `CachedRoutes` - A list of `Routes` to proxy with caching.
* `PassThroughRoutes` - A list of `Routes` to proxy without caching.

A `Route` is defined as follows:

* `Methods` - A list of HTTP methods that should be proxied
* `Pattern` - A path pattern to match. Patterns are defined per the [httprouter](https://github.com/julienschmidt/httprouter) library.

All requests that do not match a proxied route will return a `404` error response.

To build, simply `docker build --tag=test-api .` from the `_api` directory or see `docker-compose` sample referenced
below.

## `_api` project

This is a simple dummy API backed by an in-memory map used for testing purposes. This API is "secured" via a Basic Auth
username/password that is specified via the `USERNAME` and `PASSWORD` environment variables. This is obviously not a
production-worthy authentication solution, but it is sufficient to verify that the Authorization header is passed
through to the API and that requests are cached on a per-account basis.

To build, simply `docker build --tag=test-api .` from the `_api` directory or see `docker-compose` sample referenced 
below.

## Docker Compose

A sample `docker-compose.yml` file is provided for testing. This compose file contains a proxy that is configured to
proxy all routes supported by the `_api` project. HTTP `GET` requests to `/db/documents/{id}` and `/db/documents/query`
  are cached. `POST` requests to `/db/documents` and `DELETE` requests to `/db/documents/{id}` are not cached.

To build and run the docker-compose file, simply have `docker-compose` installed and run:

```
$  docker-compose up --build
Building proxy
Step 1/8 : FROM golang:1.15 AS builder
...
Successfully built a63e855b5ef7
Successfully tagged caching-reverse-proxy_proxy:latest
Building api
Step 1/8 : FROM golang:1.15 AS builder
...

Successfully built e337512e0250
Successfully tagged caching-reverse-proxy_api:latest
Recreating caching-reverse-proxy_proxy_1 ... done
Starting caching-reverse-proxy_api_1     ... done
Attaching to caching-reverse-proxy_api_1, caching-reverse-proxy_proxy_1
```

You can now make requests to the proxy server at `localhost:8080`, and you should see relevant log messages in your
`docker-compose` logs.

The following example uses the credentials specified for the `_api` in the `docker-compose` file:

```
$ curl --location --request POST 'http://localhost:8080/db/documents' \
--header 'Authorization: Basic dHJpc3RhbjpzZWNyZXRQYXNzd29yZA==' \
--header 'Content-Type: application/json' \
--data-raw '
  {
    "name": "Tristan Bull",
    "occupation": "Software Engineer"
  }'

$ curl --location --request GET 'http://localhost:8080/db/documents/fea23c45-6606-46b4-9dcb-543d4c12d08a' \
--header 'Authorization: Basic dHJpc3RhbjpzZWNyZXRQYXNzd29yZA==' \
--header 'Content-Type: application/json' \

$ curl --location --request GET 'http://localhost:8080/db/documents/fea23c45-6606-46b4-9dcb-543d4c12d08a' \
--header 'Authorization: Basic dHJpc3RhbjpzZWNyZXRQYXNzd29yZA==' \
--header 'Content-Type: application/json' \

$ curl --location --request GET 'http://localhost:8080/db/documents/fea23c45-6606-46b4-9dcb-543d4c12d08a' \
--header 'Content-Type: application/json'
```

results in the following log output:

```
proxy_1  | time="2021-01-30T19:09:27Z" level=info msg="PassThroughHandler: Handling POST at URL /db/documents."
api_1    | 2021/01/30 19:09:27 Authenticated user tristan
proxy_1  | time="2021-01-30T19:10:39Z" level=info msg="CachingHandler: Handling GET at URL /db/documents/fea23c45-6606-46b4-9dcb-543d4c12d08a."
proxy_1  | time="2021-01-30T19:10:39Z" level=debug msg="CachingHandler: Did not find key 004471b022eed335d17a30db76b16b97d4a55a75ac330241c5217929fba5dc22-GET-/db/documents/fea23c45-6606-46b4-9dcb-543d4c12d08a in cache. Proxying request."
api_1    | 2021/01/30 19:10:39 Authenticated user tristan
proxy_1  | time="2021-01-30T19:10:41Z" level=info msg="CachingHandler: Handling GET at URL /db/documents/fea23c45-6606-46b4-9dcb-543d4c12d08a."
proxy_1  | time="2021-01-30T19:10:41Z" level=debug msg="CachingHandler: Found key 004471b022eed335d17a30db76b16b97d4a55a75ac330241c5217929fba5dc22-GET-/db/documents/fea23c45-6606-46b4-9dcb-543d4c12d08a in cache."
proxy_1  | time="2021-01-30T19:10:54Z" level=info msg="CachingHandler: Handling GET at URL /db/documents/fea23c45-6606-46b4-9dcb-543d4c12d08a."
proxy_1  | time="2021-01-30T19:10:54Z" level=debug msg="CachingHandler: Did not find key e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855-GET-/db/documents/fea23c45-6606-46b4-9dcb-543d4c12d08a in cache. Proxying request."
```

## Benchmarking

There is a simple benchmark in the proxy class that roughly measures the time to process a request based on:

* Request size - 100, 1000, and 1000000 bytes
* Cache fullness - empty, 50%, 100% full
* Cache hit rate for incoming requests - 0%, 50%, 100%

The benchmark is executed against a 1GB cache though the size of the cache, and all of the parameters above can be
edited in the test. Sample output is as follows:

```bash
$ go test -timeout 30m -bench=. ./proxy
INFO[0000] PassThroughHandler: Handling GET at URL /api/things.
INFO[0000] PassThroughHandler: Handling GET at URL /api/things.
INFO[0000] PassThroughHandler: Handling GET at URL /api/things.
INFO[0000] PassThroughHandler: Handling GET at URL /api/things.
INFO[0000] PassThroughHandler: Handling GET at URL /api/things.
INFO[0000] CachingHandler: Handling GET at URL /api/things.
INFO[0000] CachingHandler: Handling GET at URL /api/things.
INFO[0000] CachingHandler: Handling GET at URL /api/things.
INFO[0000] CachingHandler: Handling GET at URL /api/things.
INFO[0000] CachingHandler: Handling GET at URL /api/things.
goos: darwin
goarch: amd64
pkg: github.com/tmbull/caching-reverse-proxy/proxy
BenchmarkProxy_CachingHandler/Request_Size_100_Cache_Fullness_0.200000_Hit_Rate_0.000000-16         	   15819	     86317 ns/op
BenchmarkProxy_CachingHandler/Request_Size_100_Cache_Fullness_0.200000_Hit_Rate_0.500000-16         	   31515	     40101 ns/op
BenchmarkProxy_CachingHandler/Request_Size_100_Cache_Fullness_0.200000_Hit_Rate_1.000000-16         	  309746	      3860 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000_Cache_Fullness_0.200000_Hit_Rate_0.000000-16        	   15102	     73534 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000_Cache_Fullness_0.200000_Hit_Rate_0.500000-16        	   32883	     41600 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000_Cache_Fullness_0.200000_Hit_Rate_1.000000-16        	  326510	      3891 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000000_Cache_Fullness_0.200000_Hit_Rate_0.000000-16     	    1419	   1123214 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000000_Cache_Fullness_0.200000_Hit_Rate_0.500000-16     	  239788	      4461 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000000_Cache_Fullness_0.200000_Hit_Rate_1.000000-16     	  290043	      3591 ns/op
BenchmarkProxy_CachingHandler/Request_Size_100_Cache_Fullness_0.500000_Hit_Rate_0.000000-16         	   16818	     73488 ns/op
BenchmarkProxy_CachingHandler/Request_Size_100_Cache_Fullness_0.500000_Hit_Rate_0.500000-16         	   28832	     40576 ns/op
BenchmarkProxy_CachingHandler/Request_Size_100_Cache_Fullness_0.500000_Hit_Rate_1.000000-16         	  300903	      3889 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000_Cache_Fullness_0.500000_Hit_Rate_0.000000-16        	   14276	     77928 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000_Cache_Fullness_0.500000_Hit_Rate_0.500000-16        	   31359	     40209 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000_Cache_Fullness_0.500000_Hit_Rate_1.000000-16        	  253030	      3972 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000000_Cache_Fullness_0.500000_Hit_Rate_0.000000-16     	    1402	   1241353 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000000_Cache_Fullness_0.500000_Hit_Rate_0.500000-16     	  293360	      4132 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000000_Cache_Fullness_0.500000_Hit_Rate_1.000000-16     	  323937	      3798 ns/op
BenchmarkProxy_CachingHandler/Request_Size_100_Cache_Fullness_1.000000_Hit_Rate_0.000000-16         	   16754	     81800 ns/op
BenchmarkProxy_CachingHandler/Request_Size_100_Cache_Fullness_1.000000_Hit_Rate_0.500000-16         	   26288	     58608 ns/op
BenchmarkProxy_CachingHandler/Request_Size_100_Cache_Fullness_1.000000_Hit_Rate_1.000000-16         	   47733	     21871 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000_Cache_Fullness_1.000000_Hit_Rate_0.000000-16        	   13255	     85357 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000_Cache_Fullness_1.000000_Hit_Rate_0.500000-16        	   26360	     46424 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000_Cache_Fullness_1.000000_Hit_Rate_1.000000-16        	  247626	      4193 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000000_Cache_Fullness_1.000000_Hit_Rate_0.000000-16     	    1584	   1236412 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000000_Cache_Fullness_1.000000_Hit_Rate_0.500000-16     	  294157	      4131 ns/op
BenchmarkProxy_CachingHandler/Request_Size_1000000_Cache_Fullness_1.000000_Hit_Rate_1.000000-16     	  213274	      4952 ns/op
PASS
ok  	github.com/tmbull/caching-reverse-proxy/proxy	1379.427s
```

Some interesting notes based on the test results:

* Operation time drops as hit rate increases
  * This is expected behavior since this is the entire reason for 
caching
* Large requests (~1MB) have poor performance
  * This is especially noticeable for low cache rate tests indicating the 
time to generate the response may actually be factoring into the test results.
  * It would be worth looking for unnecessary memory allocations
* A 100% full cache seems to have slightly worse performance in some cases
  * This is likely due to the fact that records are being evicted from the cache synchronously as part of the request 
  processing

## TODOs
* [ ] Use more sophisticated cache sizing estimation (currently we do do not account for key size or the overhead of the
  underlying data structures)
* [ ] Full performance evaluation with different request sizes, traffic patterns, etc.
* [ ] Parallel performance evaluation
* [ ] Investigate migrating to [fasthttp](https://github.com/valyala/fasthttp)
* [ ] Investigate using a caching library such as [ristretto](https://github.com/dgraph-io/ristretto)