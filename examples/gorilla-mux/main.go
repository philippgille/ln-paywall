package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/philippgille/ln-paywall/pay"
)

func PingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "pong")
}

func main() {
	r := mux.NewRouter()

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
	r.Use(pay.NewHandlerMiddleware(invoiceOptions, lndOptions, redisClient))

	r.HandleFunc("/ping", PingHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8080", r))
}
