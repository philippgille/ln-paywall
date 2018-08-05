package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/philippgille/ln-paywall/pay"
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
	// Configure middleware
	invoiceOptions := pay.DefaultInvoiceOptions // Price: 1 Satoshi; Memo: "API call"
	lndOptions := pay.DefaultLNDoptions         // Address: "localhost:10009", CertFile: "tls.cert", MacaroonFile: "invoice.macaroon"
	storageClient := pay.NewGoMap()
	// Create function that we can use in the middleware chain
	withPayment := pay.NewHandlerFuncMiddleware(invoiceOptions, lndOptions, storageClient)

	// Use a chain of middlewares for the "/ping" endpoint
	http.HandleFunc("/ping", withLogging(withPayment(pongHandler)))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
