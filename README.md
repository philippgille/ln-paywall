go-paypercall
=============

Go middleware for monetizing your API on a pay-per-call basis with Bitcoin and Lightning ⚡️

We're first focusing on `net/http`-compatible middleware and later adding middleware for [Gin](https://github.com/gin-gonic/gin), [Echo](https://github.com/labstack/echo) and other popular Go web frameworks.

An API gateway is on the roadmap as well, which you can use to monetize your API that's written in *any* language, so no need to use Go.

Usage
-----

As an example we create a server that responds to requests to `/ping` with "pong". To show how to chain the middleware, the example includes a logging middleware as well.

```go
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
	withPayment := pay.NewMiddleware(satoshis, address, certFile, macaroonFile)

	// Use a chain of middlewares for the "/ping" endpoint
	http.HandleFunc("/ping", withLogging(withPayment(pongHandler)))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Related projects
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
