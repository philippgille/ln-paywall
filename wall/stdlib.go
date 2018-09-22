package wall

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/philippgille/ln-paywall/ln"
)

// NewHandlerFuncMiddleware returns a function which you can use within an http.HandlerFunc chain.
func NewHandlerFuncMiddleware(invoiceOptions InvoiceOptions, lnClient LNclient, storageClient StorageClient) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return createHandlerFunc(invoiceOptions, lnClient, storageClient, next, "HandlerFunc")
	}
}

// NewHandlerMiddleware returns a function which you can use within an http.Handler chain.
func NewHandlerMiddleware(invoiceOptions InvoiceOptions, lnClient LNclient, storageClient StorageClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(createHandlerFunc(invoiceOptions, lnClient, storageClient, next.ServeHTTP, "Handler"))
	}
}

func createHandlerFunc(invoiceOptions InvoiceOptions, lnClient LNclient, storageClient StorageClient, next http.HandlerFunc, handlingType string) func(w http.ResponseWriter, r *http.Request) {
	invoiceOptions = assignDefaultValues(invoiceOptions)
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the request contains a header with the preimage that we need to check if the requester paid
		preimageHex := r.Header.Get("x-preimage")
		if preimageHex == "" {
			// Generate the invoice
			invoice, err := lnClient.GenerateInvoice(invoiceOptions.Price, invoiceOptions.Memo)
			if err != nil {
				errorMsg := fmt.Sprintf("Couldn't generate invoice: %+v", err)
				log.Println(errorMsg)
				http.Error(w, errorMsg, http.StatusInternalServerError)
			} else {
				// Cache the invoice metadata
				invoiceID := hex.EncodeToString([]byte(invoice.PaymentHash))
				metadata := invoiceMetaData{
					Method: r.Method,
					Path:   r.URL.Path,
				}
				storageClient.Set(invoiceID, metadata)

				stdOutLogger.Printf("Sending invoice in response: %v", invoice)
				// Note: w.Header().Set(...) must be called before w.WriteHeader(...)!
				w.Header().Set("Content-Type", "application/vnd.lightning.bolt11")
				w.WriteHeader(http.StatusPaymentRequired)
				// The actual invoice goes into the body
				w.Write([]byte(invoice.PaymentRequest))
			}
		} else {
			// Check if the provided preimage belongs to a settled API payment invoice and that it wasn't already used. Also store used preimages.
			invalidPreimageMsg, err := handlePreimage(r, storageClient, lnClient)
			if err != nil {
				errorMsg := fmt.Sprintf("An error occurred during checking the preimage: %+v", err)
				log.Printf("%v\n", errorMsg)
				http.Error(w, errorMsg, http.StatusInternalServerError)
			} else if invalidPreimageMsg != "" {
				log.Printf("%v: %v\n", invalidPreimageMsg, preimageHex)
				http.Error(w, invalidPreimageMsg, http.StatusBadRequest)
			} else {
				preimageHash, err := ln.HashPreimage(preimageHex)
				if err == nil {
					stdOutLogger.Printf("The provided preimage is valid. Continuing to the next %v. Preimage hash: %v\n", handlingType, preimageHash)
				}
				next.ServeHTTP(w, r)
			}
		}
	}
}
