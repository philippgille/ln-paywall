ln-paywall
==========

[![GoDoc](http://www.godoc.org/github.com/philippgille/ln-paywall/pay?status.svg)](http://www.godoc.org/github.com/philippgille/ln-paywall/pay) [![Build Status](https://travis-ci.org/philippgille/ln-paywall.svg?branch=master)](https://travis-ci.org/philippgille/ln-paywall) [![Go Report Card](https://goreportcard.com/badge/github.com/philippgille/ln-paywall)](https://goreportcard.com/report/github.com/philippgille/ln-paywall) [![GitHub Releases](https://img.shields.io/github/release/philippgille/ln-paywall.svg)](https://github.com/philippgille/ln-paywall/releases)

Go middleware for monetizing your API on a per-request basis with Bitcoin and Lightning ⚡️

Middlewares for:

- [X] [net/http](https://golang.org/pkg/net/http/) `HandlerFunc`
- [X] [net/http](https://golang.org/pkg/net/http/) `Handler`
	- Compatible with routers like [gorilla/mux](https://github.com/gorilla/mux) and [chi](https://github.com/go-chi/chi)
- [X] [Gin](https://github.com/gin-gonic/gin)
- [X] [Echo](https://github.com/labstack/echo)

An API gateway is on the roadmap as well, which you can use to monetize your API that's written in *any* language, not just in Go.

Contents
--------

- [Purpose](#purpose)
- [How it works](#how-it-works)
- [Prerequisites](#prerequisites)
- [Usage](#usage)
    - [net/http HandlerFunc](#nethttp-HandlerFunc)
    - [Gin](#gin)
    - [gorilla/mux](#gorillamux)
- [Related projects](#related-projects)

Purpose
-------

Until the rise of cryptocurrencies, if you wanted to monetize your API (set up a paywall), you had to:

1. Use a centralized service (like PayPal)
    - Can shut you down any time
    - High fees
    - Your API users need an account
    - Can be hacked
2. Keep track of your API users (keep accounts and their API keys in some database)
    - Privacy concerns
    - Data breaches / leaks
3. Charge for a bunch of requests, like 10.000 at a time, because real per-request payments weren't possible

With cryptocurrencies in general some of those problems were solved, but with long confirmation times and per-transaction fees a real per-request billing was still not feasable.

But then came the [Lightning Network](https://lightning.network/), an implementation of routed payment channels, which enables *real* **instant microtransactions** with **extremely low fees** (which cryptocurrencies have long promised, but never delivered). It's a *second layer* on top of existing cryptocurrencies like Bitcoin that scales far beyond the limitations of the underlying blockchain.

`ln-paywall` makes it easy to set up an API paywall for payments over the Lightning Network.

How it works
------------

With `ln-paywall` you can simply use one of the provided middlewares in your Go web service to have your web service do two things:

1. The first request gets rejected with the `402 Payment Required` HTTP status, a `Content-Type: application/vnd.lightning.bolt11` header and a Lightning ([BOLT-11](https://github.com/lightningnetwork/lightning-rfc/blob/master/11-payment-encoding.md)-conforming) invoice in the body
2. The second request must contain a `X-Preimage` header with the preimage of the paid Lightning invoice. The middleware checks if 1) the invoice was paid and 2) not already used for a previous request. If both preconditions are met, it continues to the next middleware or final request handler.

Prerequisites
-------------

There are currently two prerequisites:

1. A running [lnd](https://github.com/lightningnetwork/lnd) node which listens to gRPC connections
	- If you don't run it locally, it needs to listen to connections from external machines (so for example on 0.0.0.0 instead of localhost) and has the TLS certificate configured to include the external IP address of the node.
2. A supported storage mechanism. It's used to cache preimages that have been used as a payment for an API call, so that a user can't do multiple requests with the same preimage of a settled Lightning payment. The `pay` package currently provides factory functions for the following storages:
	- [Redis](https://redis.io/)
		- Run for example with Docker: `docker run -d redis`
			- Note: In production you should use a configuration with password!
	- A simple Go map
		- Disadvantage: Doesn't persist data, so when you restart your server, users can re-use old preimages
	- Roll your own!
		- Just implement the simple `ln.StorageClient` interface (only two methods!)

Usage
-----

Get the package with `go get -u github.com/philippgille/ln-paywall/...`.

We strongly encourage you to use vendoring, because as long as `ln-paywall` is version `0.x`, breaking changes may be introduced in new versions, including changes to the package name / import path. The project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html) and all notable changes to this project are documented in [RELEASES.md](https://github.com/philippgille/ln-paywall/blob/master/RELEASES.md).

The best way to see how to use `ln-paywall` is by example. In the below examples we create a web service that responds to requests to `/ping` with "pong".

### Gin

```Go
package main

import (
	"github.com/gin-gonic/gin"

	"github.com/philippgille/ln-paywall/pay"
)

func main() {
	r := gin.Default()

	// Configure and use middleware
	invoiceOptions := pay.InvoiceOptions{
		Amount: 1,
		Memo:   "Ping API call",
	}
	lndOptions := pay.LNDoptions{
		Address:      "123.123.123.123:10009",
		CertFile:     "tls.cert",
		MacaroonFile: "invoice.macaroon",
	}
	redisClient := pay.NewRedisClient(pay.DefaultRedisOptions) // Connects to localhost:6379
	r.Use(pay.NewGinMiddleware(invoiceOptions, lndOptions, redisClient))

	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	
	r.Run() // listen and serve on 0.0.0.0:8080
}
```

### gorilla/mux

See [example](examples/gorilla-mux/main.go).

### net/http HandlerFunc

To show how to chain the middleware, this example includes a logging middleware as well.

See [example](examples/handlerfunc/main.go).

### Echo

See [example](examples/echo/main.go).

Related Projects
----------------

- [https://github.com/ElementsProject/paypercall](https://github.com/ElementsProject/paypercall)
    - Middleware for the JavaScript web framework [Express](https://expressjs.com/)
    - Reverse proxy
    - Payment: Lightning Network
- [https://github.com/interledgerjs/koa-web-monetization](https://github.com/interledgerjs/koa-web-monetization)
    - Middleware for the JavaScript web framework [Koa](https://koajs.com/)
    - Payment: Interledger
- [https://moonbanking.com/api](https://moonbanking.com/api)
    - API that *uses* a similar functionality, not *providing* it
- [https://www.coinbee.io/](https://www.coinbee.io/)
	- Paid service for Bitcoin paywalls (no Lightning)
	- Looks like its meant for websites only, not APIs, because the browser session is involved
