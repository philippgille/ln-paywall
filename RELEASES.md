Releases
========

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

vNext
-----

v0.2.0 (2018-07-29)
-------------------

Enhancements:

- Added: Interface `pay.StorageClient` - an abstraction for multiple storage clients, which allows you to write your own storage client for storing the preimages that have already been used as payment proof in a request
    - Methods `WasUsed(string) (bool, error)` and `SetUsed(string) error`
- Added: Struct `pay.RedisClient` - implements the `StorageClient` interface
    - Factory function `NewRedisClient(...)`
- Added: Var `pay.DefaultRedisOptions` - a `Redisoptions` object with default values
- Added: Struct `pay.GoMap` - implements the `StorageClient` interface
    - Factory function `NewGoMap()`
- Improved: Increased middleware performance and decreased load on the connected lnd when invalid requests with the `x-preimage` header are received (invalid because the preimage was already used) - Instead of first getting a corresponding invoice for a preimage from the lnd and *then* checking if the preimage was used already, the order of these operations was switched, because then, if the preimage was already used, no request to lnd needs to be made anymore.
- Improved: All fields of the struct `pay.RedisOptions` are now optional

Breaking changes:

- Changed: `pay.NewHandlerFuncMiddleware(...)`, `pay.NewHandlerMiddleware(...)` and `pay.NewGinMiddleware(...)` now take a `ln.StorageClient` instead of a `*redis.Client` as parameter
- Changed: `ln.CheckPreimage(...)` was renamed to `ln.CheckInvoice(...)` and doesn't check the storage anymore. The `ln` methods are supposed to just handle lightning related things and nothing else. This shouldn't affect anyone, because `ln` is meant to be used only internally.
- Removed: Package `lnrpc` - Instead of using our own generated lnd gRPC Go file, import the one from Lightning Labs. This shouldn't affect anyone, because it was meant to be used only internally.

v0.1.0 (2018-07-02)
-------------------

Initial release after working on the project during the "Chainhack 3" [Blockchain hackathon](https://blockchain-hackathon.com/).

- Go middlewares for:
    - [net/http](https://golang.org/pkg/net/http/) `HandlerFunc`
    - [net/http](https://golang.org/pkg/net/http/) `Handler`
        - Compatible with routers like [gorilla/mux](https://github.com/gorilla/mux) and [chi](https://github.com/go-chi/chi)
    - [Gin](https://github.com/gin-gonic/gin)
- Supported Lightning Network node:
    - [lnd](https://github.com/lightningnetwork/lnd)
- Supported storage:
    - [Redis](https://redis.io/)
