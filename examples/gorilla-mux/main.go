package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/philippgille/ln-paywall/storage"
	"github.com/philippgille/ln-paywall/wall"
)

func PingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "pong")
}

func main() {
	r := mux.NewRouter()

	// Configure and use middleware
	invoiceOptions := wall.DefaultInvoiceOptions // Price: 1 Satoshi; Memo: "API call"
	lndOptions := wall.DefaultLNDoptions         // Address: "localhost:10009", CertFile: "tls.cert", MacaroonFile: "invoice.macaroon"
	storageClient := storage.NewGoMap()          // Local in-memory cache
	r.Use(wall.NewHandlerMiddleware(invoiceOptions, lndOptions, storageClient))

	r.HandleFunc("/ping", PingHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8080", r))
}
