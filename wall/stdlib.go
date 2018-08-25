package wall

import (
	"fmt"
	"log"
	"net/http"

	"github.com/philippgille/ln-paywall/ln"
)

// NewHandlerFuncMiddleware returns a function which you can use within an http.HandlerFunc chain.
func NewHandlerFuncMiddleware(invoiceOptions InvoiceOptions, lndOptions LNDoptions, storageClient StorageClient) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return createHandlerFunc(invoiceOptions, lndOptions, storageClient, next, "HandlerFunc")
	}
}

// NewHandlerMiddleware returns a function which you can use within an http.Handler chain.
func NewHandlerMiddleware(invoiceOptions InvoiceOptions, lndOptions LNDoptions, storageClient StorageClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(createHandlerFunc(invoiceOptions, lndOptions, storageClient, next.ServeHTTP, "Handler"))
	}
}

func createHandlerFunc(invoiceOptions InvoiceOptions, lndOptions LNDoptions, storageClient StorageClient, next http.HandlerFunc, handlingType string) func(w http.ResponseWriter, r *http.Request) {
	invoiceOptions, lndOptions = assignDefaultValues(invoiceOptions, lndOptions)
	client, err := ln.NewLNDclient(lndOptions.Address, lndOptions.CertFile, lndOptions.MacaroonFile)
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the request contains a header with the preimage that we need to check if the requester paid
		preimage := r.Header.Get("x-preimage")
		if preimage == "" {
			// Generate the invoice
			invoice, err := client.GenerateInvoice(invoiceOptions.Price, invoiceOptions.Memo)
			if err != nil {
				errorMsg := fmt.Sprintf("Couldn't generate invoice: %+v", err)
				log.Println(errorMsg)
				http.Error(w, errorMsg, http.StatusInternalServerError)
			} else {
				stdOutLogger.Printf("Sending invoice in response: %v", invoice)
				// Note: w.Header().Set(...) must be called before w.WriteHeader(...)!
				w.Header().Set("Content-Type", "application/vnd.lightning.bolt11")
				w.WriteHeader(http.StatusPaymentRequired)
				// The actual invoice goes into the body
				w.Write([]byte(invoice))
			}
		} else {
			// Check if the provided preimage belongs to a settled API payment invoice and that it wasn't already used and store used preimages
			ok, err := handlePreimage(preimage, storageClient, client)
			if err != nil {
				errorMsg := fmt.Sprintf("An error occured during checking the preimage: %+v", err)
				log.Printf("%v\n", errorMsg)
				http.Error(w, errorMsg, http.StatusInternalServerError)
			} else {
				if !ok {
					log.Printf("The provided preimage is invalid: %v\n", preimage)
					http.Error(w, "The provided preimage is invalid", http.StatusBadRequest)
				} else {
					preimageHash, err := ln.HashPreimage(preimage)
					if err == nil {
						stdOutLogger.Printf("The provided preimage is valid. Continuing to the next %v. Preimage hash: %v\n", handlingType, preimageHash)
					}
					next.ServeHTTP(w, r)
				}
			}
		}
	}
}
