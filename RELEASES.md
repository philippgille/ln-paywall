Releases
========

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

vNext
-----

v0.5.1 (2018-10-02)
-------------------

- Fixed: Performance decreased when using Lightning Charge and the amount of invoices in the Lightning Charge server increased (issue [#28](https://github.com/philippgille/ln-paywall/issues/28))
- Fixed: Since the introduction of the `ln.Invoice` struct the whole struct was logged instead of just the invoice string

### Breaking changes

> Note: The following breaking changes don't affect normal users of the package, but only those who use their own implementations of our interfaces.

- Changed: The struct `ln.Invoice` now has a field `ImplDepID string` which is required by the middlewares. It's an LN node implementation dependent ID (e.g. payment hash for lnd, some random string for Lightning Charge). (Required for issue [#28](https://github.com/philippgille/ln-paywall/issues/28).)
- Changed: `wall.LNclient` now requires the method `CheckInvoice(string) (bool, error)` to accept the LN node implementation dependent ID instead of the preimage hash as parameter. (Required for issue [#28](https://github.com/philippgille/ln-paywall/issues/28).)

v0.5.0 (2018-09-24)
-------------------

- Added: Support for [c-lightning](https://github.com/ElementsProject/lightning) with [Lightning Charge](https://github.com/ElementsProject/lightning-charge) (issue [#6](https://github.com/philippgille/ln-paywall/issues/6))
    - Note: The current implementation's performance decreases when the amount of invoices in the Lightning Charge server increases. This will be fixed in an upcoming release.
- Added: Package `pay` (issue [#20](https://github.com/philippgille/ln-paywall/issues/20))
    - Interface `pay.LNclient` - Abstraction of a Lightning Network node client for paying LN invoices. Enables developers to write their own implementations if the provided ones aren't enough.
    - Struct `pay.Client` - Replacement for a standard Go `http.Client`
        - Factory function `NewClient(httpClient *http.Client, lnClient LNclient) Client` - `httpClient` can be passed as `nil`, leading to `http.DefaultClient` being used
        - Method `Do(req *http.Request) (*http.Response, error)` - Meant to be used as equivalent to the same Go `http.Client` method, but handles the Lightning Network payment in the background.
        - Method `Get(url string) (*http.Response, error)` - A convenience method for the `Do(...)` method, also an equivalent to the same Go `http.Client` method (regarding its usage)
    - Example client in `examples/client/main.go`
    - > Note: Currently only `ln.LNDclient` can be used in the client, because Lightning Charge doesn't support sending payments (and maybe never will)
- Added: Method `Pay(invoice string) (string, error)` for `ln.LNDclient` - Implements the new `pay.LNclient` interface, so that the `LNDclient` can be used as parameter in the `pay.NewClient(...)` function. (Issue [#20](https://github.com/philippgille/ln-paywall/issues/20))
- Fixed: A client could cheat in multiple ways, for example use a preimage for a request to endpoint A while the invoice was for endpoint B, with B being cheaper than A. Or if the LN node is used for other purposes as well, a client could send any preimage that might be for a totally different invoice, not related to the API at all. (Issue [#16](https://github.com/philippgille/ln-paywall/issues/16))
- Fixed: Some info logs were logged to `stderr` instead of `stdout`

### Breaking changes

- Changed: The preimage in the `X-Preimage` header must now be hex encoded instead of Base64 encoded. The hex encoded representation is the typical representation, as used by "lncli listpayments", Eclair on Android and bolt11 payment request decoders like [https://lndecode.com](https://lndecode.com). Base64 was used previously because "lncli listinvoices" uses that encoding. (Issue [#8](https://github.com/philippgille/ln-paywall/issues/8))
- Changed: Interface `wall.StorageClient` and thus all its implementations in the `storage` package were significantly changed. The methods are now completely independent of any ln-paywall specifics, with `Set(...)` and `Get(...)` just setting and retrieving any arbitrary `interface{}` to/from the storage. (Required for issue [#16](https://github.com/philippgille/ln-paywall/issues/16).)
- Changed: The method `GenerateInvoice(int64, string) (string, error)` in the interface `wall.LNclient` was changed to return a `ln.Invoice` object, which makes it much easier to access the preimage hash (a.k.a. payment hash), instead of having to decode the invoice. (Useful for issue [#16](https://github.com/philippgille/ln-paywall/issues/16), in which the preimage hash is required as key in the storage.)

v0.4.0 (2018-09-03)
-------------------

> Warning: This release contains a lot of renamings and refactorings, so your code will most definitely break. But it paves the way to upcoming features and makes some things easier, like automated testing.

- Added: Package-level documentation
- Improved: In case of an invalid preimage the error response is much more detailed now. It differentiates between several reasons why the preimage is invalid. Additionally more cases of invalid requests are detected now, so a proper `400 Bad Request` is returned instead of a `500 Internal Server Error`. (Issue [#11](https://github.com/philippgille/ln-paywall/issues/11))
- Improved: Increased performance when creating multiple middleware instances, because the LN client implementation can now be passed into the middleware factory function and be reused across multiple middleware instances. Previously the LN client was created internally, and a new instance was created with every middleware instance.
    - Not measured, but probably a bit lower memory consumption and a bit less traffic. Probably not much regarding speed.
- Fixed: Wrong spelling in an error message

### Breaking changes

- Changed: Renamed package from `pay` to `wall` - this enables us to create a package called `pay` for client-side payments to the paywall in the future
- Changed: Moved all storage implementations to the new package `storage`
- Changed: Moved `LNDoptions` and `DefaultLNDoptions` to the from the `wall` (former `pay`) package to the `ln` package
    - This leads to the same kind of separation and loose coupling as with the storages
- Changed: All middleware factory functions now take a `LNclient` as second parameter instead of `LNDoptions`
    - This also leads to the same kind of separation and loose coupling as with the storages
    - In addition it enables proper mocking of the LN client for tests (preparation for issue [#10](https://github.com/philippgille/ln-paywall/issues/10))
    - As well as own implementations of LN clients (preparation for issue [#6](https://github.com/philippgille/ln-paywall/issues/6))

v0.3.0 (2018-08-12)
-------------------

- Added: `pay.NewEchoMiddleware(...)` - A middleware factory function for [Echo](https://github.com/labstack/echo) (issue [#2](https://github.com/philippgille/ln-paywall/issues/2))
- Added: Bolt DB client (issue [#3](https://github.com/philippgille/ln-paywall/issues/3))
    - Struct `pay.BoltClient` - Implements the `StorageClient` interface
    - Factory function `NewBoltClient(...)`
    - Struct `pay.BoltOptions` - Options for the `BoltClient`
    - Var `pay.DefaultBoltOptions` - a `BoltOptions` object with default values
- Added: `pay.LNclient` - An abstraction of a client that connects to a Lightning Network node implementation (like lnd, c-lightning and eclair)
    - Implemented for issue [#4](https://github.com/philippgille/ln-paywall/issues/4), but will be useful for issue [#6](https://github.com/philippgille/ln-paywall/issues/6) as well
- Added: `ln.LNDclient` - Implements the `pay.LNclient` interface (issue [#4](https://github.com/philippgille/ln-paywall/issues/4))
    - Factory function `ln.NewLNDclient(...)`
- Improved: Increased middleware performance by reusing the gRPC connection to the lnd backend (issue [#4](https://github.com/philippgille/ln-paywall/issues/4))
    - With the same setup (local Gin web service, `pay.GoMap` as storage client, remote lnd, same hardware) it took about 100ms per request before, and takes about 25ms per request now. Measured from the arrival of the initial request until the sending of the response with the Lightning invoice (as logged by Gin).
- Fixed: Success log message mentioned "HandlerFunc" in all middlewares despite it not always being a HandlerFunc
- Fixed: A wrong HTTP status code was used in responses when an internal error occurred (`400 Bad Request` instead of `500 Internal Server Error`)

### Breaking changes

Package `pay`:

- Changed: Renamed `pay.InvoiceOptions.Amount` to `pay.InvoiceOptions.Price` to avoid misunderstandings

Package `ln` (none of these changes should affect anyone, because this package is meant to be used only internally):

- Removed: `ln.NewLightningClient(...)` - Not required anymore after adding the much more usable `ln.NewLNDclient(...)`.
- Changed: `ln.GenerateInvoice(...)` from being a function to being a method of `ln.LNDclient` and removed all lnd connection-related parameters which are part of the `LNDclient`. (issue [#4](https://github.com/philippgille/ln-paywall/issues/4))
- Changed `ln.CheckInvoice(...)` from being a function to being a method of `ln.LNDclient` and removed all lnd connection-related parameters which are part of the `LNDclient`. (issue [#4](https://github.com/philippgille/ln-paywall/issues/4))

v0.2.0 (2018-07-29)
-------------------

- Added: Interface `pay.StorageClient` - an abstraction for multiple storage clients, which allows you to write your own storage client for storing the preimages that have already been used as payment proof in a request (issue [#1](https://github.com/philippgille/ln-paywall/issues/1))
    - Methods `WasUsed(string) (bool, error)` and `SetUsed(string) error`
- Added: Struct `pay.RedisClient` - implements the `StorageClient` interface (issue [#1](https://github.com/philippgille/ln-paywall/issues/1))
    - Factory function `NewRedisClient(...)`
- Added: Var `pay.DefaultRedisOptions` - a `RedisOptions` object with default values
- Added: Struct `pay.GoMap` - implements the `StorageClient` interface (issue [#1](https://github.com/philippgille/ln-paywall/issues/1))
    - Factory function `NewGoMap()`
- Improved: Increased middleware performance and decreased load on the connected lnd when invalid requests with the `x-preimage` header are received (invalid because the preimage was already used) - Instead of first getting a corresponding invoice for a preimage from the lnd and *then* checking if the preimage was used already, the order of these operations was switched, because then, if the preimage was already used, no request to lnd needs to be made anymore.
- Improved: All fields of the struct `pay.RedisOptions` are now optional

### Breaking changes

Package `pay`:

- Changed: `pay.NewHandlerFuncMiddleware(...)`, `pay.NewHandlerMiddleware(...)` and `pay.NewGinMiddleware(...)` now take a `ln.StorageClient` instead of a `*redis.Client` as parameter (issue [#1](https://github.com/philippgille/ln-paywall/issues/1))

Package `ln` (none of these changes should affect anyone, because this package is meant to be used only internally):

- Changed: `ln.CheckPreimage(...)` was renamed to `ln.CheckInvoice(...)` and doesn't check the storage anymore. The `ln` methods are supposed to just handle lightning related things and nothing else.
- Removed: Package `lnrpc` - Instead of using our own generated lnd gRPC Go file, import the one from Lightning Labs.

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
