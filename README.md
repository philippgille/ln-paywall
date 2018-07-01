go-paypercall
=============

[![Go Report Card](https://goreportcard.com/badge/github.com/philippgille/go-paypercall)](https://goreportcard.com/report/github.com/philippgille/go-paypercall)

Go middleware for monetizing your API on a pay-per-call basis with Bitcoin and Lightning ⚡️

Middlewares for:

- [X] [net/http](https://golang.org/pkg/net/http/) `HandlerFunc`
- [X] [net/http](https://golang.org/pkg/net/http/) `Handler` (also compatible with routers like [gorilla/mux](https://github.com/gorilla/mux) and [chi](https://github.com/go-chi/chi))
- [X] [Gin](https://github.com/gin-gonic/gin)
- [ ] [Echo](https://github.com/labstack/echo)

An API gateway is on the roadmap as well, which you can use to monetize your API that's written in *any* language, not just in Go.

Contents
--------

- [Purpose](#purpose)
- [Usage](#usage)
    - [net/http HandlerFunc](#nethttp-HandlerFunc)
    - [Gin](#gin)
- [Related projects](#related-projects)

Purpose
-------

Until the rise of cryptocurrencies, if you wanted to monetize your API (set up a paywall), you had to:

1. Use a centralized service (like PayPal) or bank
    - Can shut you down any time
    - High fees
    - Your API users need an account
    - Can be hacked
2. Keep track of your API users (keep accounts and their API keys in some database)
    - Privacy concerns
    - Data breaches
3. Charge for a bunch of requests, like 10.000 at a time, because real per-request payments weren't possible

With cryptocurrencies in general some of those problems were solved, but with long confirmation times and per-transaction fees a real per-request billing was still not feasable.

But then came the [Lightning Network](https://lightning.network/), an implementation of routed payment channels, which enables *real* **instant** and **zero-fee** transactions (which cryptocurrencies have long promised, but never delivered).

With `go-paypercall` you can simply use one of the provided middlewares in your Go web service to have your web service do two things:

1. The first request gets rejected with the `402 Payment Required` HTTP status and a Lightning ([BOLT-11](https://github.com/lightningnetwork/lightning-rfc/blob/master/11-payment-encoding.md)-compatible) invoice in the body
2. The second request must contain a `X-preimage` header with the preimage of the paid Lightning invoice. The middleware checks if the invoice was paid and if yes, continues to the next middleware in your middleware chain.

Usage
-----

The only prerequisite is a running [lnd](https://github.com/lightningnetwork/lnd) node which listens to gRPC connections from external machines (so for example on 0.0.0.0 instead of localhost) and has the TLS certificate configured to include the external IP address of the node.

In both examples we create a web server that responds to requests to `/ping` with "pong".

### net/http HandlerFunc

To show how to chain the middleware, this example includes a logging middleware as well.

```Go
package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/philippgille/go-paypercall/pay"
)

func pongHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "pong")
}

func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Logged connection from %s", r.RemoteAddr)
		next.ServeHTTP(w, r)
	}
}

func main() {
	certFile, err := filepath.Abs("tls.cert")
	if err != nil {
		panic(err)
	}
	macaroonFile, err := filepath.Abs("invoice.macaroon")
	if err != nil {
		panic(err)
	}
	satoshis := int64(1)
	address := "123.123.123.123:10009"
	// Create function that we can use in the middleware chain
	withPayment := pay.NewHandlerFuncMiddleware(satoshis, address, certFile, macaroonFile)

	// Use a chain of middlewares for the "/ping" endpoint
	http.HandleFunc("/ping", withLogging(withPayment(pongHandler)))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Gin

```Go
package main

import (
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/philippgille/go-paypercall/pay"
)

func main() {
	r := gin.Default()

	certFile, err := filepath.Abs("tls.cert")
	if err != nil {
		panic(err)
	}
	macaroonFile, err := filepath.Abs("invoice.macaroon")
	if err != nil {
		panic(err)
	}
	satoshis := int64(1)
	address := "123.123.123.123:10009"

	r.Use(pay.NewGinMiddleware(satoshis, address, certFile, macaroonFile))
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	r.Run() // listen and serve on 0.0.0.0:8080
}
```

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
