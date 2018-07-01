go-paypercall
=============

[![Go Report Card](https://goreportcard.com/badge/github.com/philippgille/go-paypercall)](https://goreportcard.com/report/github.com/philippgille/go-paypercall)

Go middleware for monetizing your API on a pay-per-call basis with Bitcoin and Lightning ⚡️

Middlewares for:

- [X] [net/http](https://golang.org/pkg/net/http/) `HandlerFunc`
- [X] [net/http](https://golang.org/pkg/net/http/) `Handler` (also compatible with routers like [gorilla/mux](https://github.com/gorilla/mux) and [chi](https://github.com/go-chi/chi))
- [X] [Gin](https://github.com/gin-gonic/gin)
- [ ] [Echo](https://github.com/labstack/echo)

An API gateway is on the roadmap as well, which you can use to monetize your API that's written in *any* language, so no need to use Go.

Contents
--------

- [Usage](#usage)
    - [net/http HandlerFunc](#nethttp-HandlerFunc)
    - [Gin](#gin)
- [Related projects](#related-projects)

Usage
-----

### net/http HandlerFunc

As an example we create a server that responds to requests to `/ping` with "pong". To show how to chain the middleware, the example includes a logging middleware as well.

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
