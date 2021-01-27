# caching-reverse-proxy

This is a reverse proxy that optionally caches the responses from the backend service.


## Main project (root directory)

This is the reverse proxy. It is configurable via a JSON file (see sample [here](config.json)). Configuration options
are as follows:

* `ListenAddr` - The `host:port` to listen on.
* `TargetUrl` - The base URL to target proxied requests to.
* `CachedRoutes` - A list of `Routes` to proxy with caching.
* `PassThroughRoutes` - A list of `Routes` to proxy without caching.

A `Route` is defined as follows:

* `Methods` - A list of HTTP methods that should be proxied
* `Pattern` - A path pattern to match. Patterns are defined per the [httprouter](https://github.com/julienschmidt/httprouter) library.

All requests that do not match a proxied route will return a `404` error response.

To build, simply `docker build --tag=test-api .` from the `_api` directory or see `docker-compose` sample referenced
below.

## `_api` project

This is a simple dummy API backed by an in-memory map used for testing purposes.

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
 ---> 3360fba69704
Step 2/8 : RUN mkdir /app
 ---> Using cache
 ---> 586fa0d7598f
Step 3/8 : ADD . /app
 ---> cc5b2ff510eb
Step 4/8 : WORKDIR /app
 ---> Running in 39dae1eeb9b6
Removing intermediate container 39dae1eeb9b6
 ---> 7b0d2d61e860
Step 5/8 : RUN CGO_ENABLED=0 GOOS=linux go build -o main ./...
 ---> Running in a51881522c6d
go: downloading github.com/julienschmidt/httprouter v1.3.0
go: downloading github.com/tkanos/gonfig v0.0.0-20210106201359-53e13348de2f
go: downloading github.com/ghodss/yaml v1.0.0
go: downloading gopkg.in/yaml.v2 v2.4.0
Removing intermediate container a51881522c6d
 ---> 84a9057c66eb

Step 6/8 : FROM alpine:latest
 ---> 7731472c3f2a
Step 7/8 : COPY --from=builder /app .
 ---> 947b7c456123
Step 8/8 : CMD ["./main"]
 ---> Running in 66ba35d864f2
Removing intermediate container 66ba35d864f2
 ---> a63e855b5ef7

Successfully built a63e855b5ef7
Successfully tagged caching-reverse-proxy_proxy:latest
Building api
Step 1/8 : FROM golang:1.15 AS builder
 ---> 3360fba69704
Step 2/8 : RUN mkdir /app
 ---> Using cache
 ---> 586fa0d7598f
Step 3/8 : ADD . /app
 ---> Using cache
 ---> 1ffdafea01a4
Step 4/8 : WORKDIR /app
 ---> Using cache
 ---> 21275c6dc77a
Step 5/8 : RUN CGO_ENABLED=0 GOOS=linux go build -o main ./...
 ---> Using cache
 ---> df8b29bfad23

Step 6/8 : FROM alpine:latest
 ---> 7731472c3f2a
Step 7/8 : COPY --from=builder /app .
 ---> Using cache
 ---> 9c8a6d76cf2b
Step 8/8 : CMD ["./main"]
 ---> Using cache
 ---> e337512e0250

Successfully built e337512e0250
Successfully tagged caching-reverse-proxy_api:latest
Recreating caching-reverse-proxy_proxy_1 ... done
Starting caching-reverse-proxy_api_1     ... done
Attaching to caching-reverse-proxy_api_1, caching-reverse-proxy_proxy_1
```

You can now make requests to the proxy server at `localhost:8080`, and press `ctrl+C` when you are done testing:

```
proxy_1  | 2021/01/27 02:22:51 /db/documents
proxy_1  | 2021/01/27 02:22:58 /db/documents/31632d50-3734-4ef8-a290-2313779bdbb0
proxy_1  | 2021/01/27 02:23:01 /db/documents/31632d50-3734-4ef8-a290-2313779bdbb0
^CGracefully stopping... (press Ctrl+C again to force)
Stopping caching-reverse-proxy_proxy_1   ... done
Stopping caching-reverse-proxy_api_1     ... done
```