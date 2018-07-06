package pay

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/philippgille/ln-paywall/ln"
)

// stdOutLogger logs to stdout, while the default log package loggers log to stderr.
var stdOutLogger = log.New(os.Stdout, "", log.LstdFlags)

// InvoiceOptions are the options for an invoice.
type InvoiceOptions struct {
	// Amount of Satoshis you want to have paid for one API call
	Amount int64
	// Note to be shown on the invoice,
	// for example: "API call to api.example.com".
	// Optional ("API call" by default)
	Memo string
}

// LNDoptions are the options for the connection to the lnd node.
type LNDoptions struct {
	// Address of your LND node, including the port
	Address string
	// Path to the "tls.cert" file that your LND node uses
	CertFile string
	// Path to the "invoice.macaroon" file that your LND node uses
	MacaroonFile string
}

// NewHandlerFuncMiddleware returns a function which you can use within an http.HandlerFunc chain.
func NewHandlerFuncMiddleware(invoiceOptions InvoiceOptions, lndOptions LNDoptions, storageClient StorageClient) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return createHandlerFunc(invoiceOptions, lndOptions, storageClient, next)
	}
}

// NewHandlerMiddleware returns a function which you can use within an http.Handler chain.
func NewHandlerMiddleware(invoiceOptions InvoiceOptions, lndOptions LNDoptions, storageClient StorageClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(createHandlerFunc(invoiceOptions, lndOptions, storageClient, next.ServeHTTP))
	}
}

func createHandlerFunc(invoiceOptions InvoiceOptions, lndOptions LNDoptions, storageClient StorageClient, next http.HandlerFunc) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the request contains a header with the preimage that we need to check if the requester paid
		preimage := r.Header.Get("x-preimage")
		if preimage == "" {
			// Generate the invoice
			invoice, err := ln.GenerateInvoice(invoiceOptions.Amount, invoiceOptions.Memo,
				lndOptions.Address, lndOptions.CertFile, lndOptions.MacaroonFile)
			if err != nil {
				errorMsg := fmt.Sprintf("Couldn't generate invoice: %+v", err)
				log.Println(errorMsg)
				http.Error(w, errorMsg, http.StatusBadRequest)
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
			ok, err := handlePreimage(preimage, lndOptions.Address, lndOptions.CertFile, lndOptions.MacaroonFile, storageClient)
			if err != nil {
				errorMsg := fmt.Sprintf("An error occured during checking the preimage: %+v", err)
				log.Printf("%v\n", errorMsg)
				http.Error(w, errorMsg, http.StatusBadRequest)
			} else {
				if !ok {
					log.Printf("The provided preimage is invalid: %v\n", preimage)
					http.Error(w, "The provided preimage is invalid", http.StatusBadRequest)
				} else {
					preimageHash, err := ln.HashPreimage(preimage)
					if err == nil {
						stdOutLogger.Printf("The provided preimage is valid. Continuing to the next HandlerFunc. Preimage hash: %v\n", preimageHash)
					}
					next.ServeHTTP(w, r)
				}
			}
		}
	}
}

// NewGinMiddleware returns a Gin middleware in the form of a gin.HandlerFunc.
func NewGinMiddleware(invoiceOptions InvoiceOptions, lndOptions LNDoptions, storageClient StorageClient) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Check if the request contains a header with the preimage that we need to check if the requester paid
		preimage := ctx.GetHeader("x-preimage")
		if preimage == "" {
			// Generate the invoice
			invoice, err := ln.GenerateInvoice(invoiceOptions.Amount, invoiceOptions.Memo,
				lndOptions.Address, lndOptions.CertFile, lndOptions.MacaroonFile)
			if err != nil {
				errorMsg := fmt.Sprintf("Couldn't generate invoice: %+v", err)
				log.Println(errorMsg)
				http.Error(ctx.Writer, errorMsg, http.StatusBadRequest)
				ctx.Abort()
			} else {
				stdOutLogger.Printf("Sending invoice in response: %v", invoice)
				// Note: w.Header().Set(...) must be called before w.WriteHeader(...)!
				ctx.Header("Content-Type", "application/vnd.lightning.bolt11")
				ctx.Status(http.StatusPaymentRequired)
				// The actual invoice goes into the body
				ctx.Writer.Write([]byte(invoice))
				ctx.Abort()
			}
		} else {
			// Check if the provided preimage belongs to a settled API payment invoice and that it wasn't already used and store used preimages
			ok, err := handlePreimage(preimage, lndOptions.Address, lndOptions.CertFile, lndOptions.MacaroonFile, storageClient)
			if err != nil {
				errorMsg := fmt.Sprintf("An error occured during checking the preimage: %+v", err)
				log.Printf("%v\n", errorMsg)
				http.Error(ctx.Writer, errorMsg, http.StatusBadRequest)
				ctx.Abort()
			} else {
				if !ok {
					log.Printf("The provided preimage is invalid: %v\n", preimage)
					http.Error(ctx.Writer, "The provided preimage is invalid", http.StatusBadRequest)
					ctx.Abort()
				} else {
					preimageHash, err := ln.HashPreimage(preimage)
					if err == nil {
						stdOutLogger.Printf("The provided preimage is valid. Continuing to the next HandlerFunc. Preimage hash: %v\n", preimageHash)
					}
					ctx.Next()
				}
			}
		}
	}
}

// StorageClient is an abstraction for different storage client implementations.
// A storage client must only be able to check if a preimage was already used for a payment bofore
// and to store a preimage that was used before.
type StorageClient interface {
	WasUsed(string) (bool, error)
	SetUsed(string) error
}

// handlePreimage does four things:
// 1) Checks if the preimage was already used as a payment proof before.
// 2) Checks if the preimage corresponds to an existing invoice on the connected lnd.
// 3) Checks if the corresponding invoice was settled.
// 4) Store the preimage to the storage for future checks.
// Returns false and an empty error if the preimage was already used or if the corresponding invoice isn't settled.
// Returns true and an empty error if everythings's fine.
func handlePreimage(preimage string, address string, certFile string, macaroonFile string, storageClient StorageClient) (bool, error) {
	// Check if it was already used before
	wasUsed, err := storageClient.WasUsed(preimage)
	if err != nil {
		return false, err
	}
	if wasUsed {
		// Key was found, which means the payment was already used for an API call.
		return false, nil
	}

	// Check if a corresponding invoice exists and is settled
	settled, err := ln.CheckInvoice(preimage, address, certFile, macaroonFile)
	if err != nil {
		return false, err
	}
	if !settled {
		return false, nil
	}

	// Key not found, so it wasn't used before.
	// Insert key for future checks.
	err = storageClient.SetUsed(preimage)
	if err != nil {
		return true, err
	}
	return true, nil
}
