package pay

import (
	"fmt"
	"log"
	"net/http"

	"github.com/philippgille/go-paypercall/ln"
)

// NewMiddleware returns a function which you can use within an http.HandlerFunc chain.
// The amount parameter is the amount of satoshis you want to have paid for one API call.
// The address parameter is the address of your LND node, including the port.
// The certFile parameter is the path to the "tls.cert" file that your LND node uses.
// The macaroonFile parameter is the path to the "admin.macaroon" file that your LND node uses.
func NewHandlerFuncMiddleware(amount int64, address string, certFile string, macaroonFile string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Check if the request contains a header with the preimage that we need to check if the requester paid
			preimage := r.Header.Get("x-preimage")
			if preimage == "" {
				// Generate the invoice
				invoice, err := ln.GenerateInvoice(amount, address, certFile, macaroonFile)
				if err != nil {
					errorMsg := fmt.Sprintf("Couldn't generate invoice: %+v", err)
					log.Println(errorMsg)
					http.Error(w, errorMsg, http.StatusBadRequest)
				} else {
					// Note: w.Header().Set(...) must be called before w.WriteHeader(...)!
					w.Header().Set("Content-Type", "application/vnd.lightning.bolt11")
					w.WriteHeader(http.StatusPaymentRequired)
					// The actual invoice goes into the body
					w.Write([]byte(invoice))
				}
			} else {
				// Check if the provided preimage belongs to a settled API payment invoice and that it wasn't already used
				ok, err := ln.CheckPreimage(preimage, address, certFile, macaroonFile)
				if err != nil {
					errorMsg := fmt.Sprintf("An error occured during checking the preimage: %+v", err)
					log.Printf("%v\n", errorMsg)
					http.Error(w, errorMsg, http.StatusBadRequest)
				} else {
					if !ok {
						log.Printf("The provided preimage is invalid: %v\n", preimage)
						http.Error(w, "The provided preimage is invalid", http.StatusBadRequest)
					} else {
						next.ServeHTTP(w, r)
					}
				}
			}
		}
	}
}
