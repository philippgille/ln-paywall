package wall

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/philippgille/ln-paywall/ln"
)

// stdOutLogger logs to stdout, while the default log package loggers log to stderr.
var stdOutLogger = log.New(os.Stdout, "", log.LstdFlags)

// InvoiceOptions are the options for an invoice.
type InvoiceOptions struct {
	// Amount of Satoshis you want to have paid for one API call.
	// Values below 1 are automatically changed to the default value.
	// Optional (1 by default).
	Price int64
	// Note to be shown on the invoice,
	// for example: "API call to api.example.com".
	// Optional ("" by default).
	Memo string
}

// DefaultInvoiceOptions provides default values for InvoiceOptions.
var DefaultInvoiceOptions = InvoiceOptions{
	Price: 1,
	Memo:  "API call",
}

// LNDoptions are the options for the connection to the lnd node.
type LNDoptions struct {
	// Address of your LND node, including the port.
	// Optional ("localhost:10009" by default).
	Address string
	// Path to the "tls.cert" file that your LND node uses.
	// Optional ("tls.cert" by default).
	CertFile string
	// Path to the "invoice.macaroon" file that your LND node uses.
	// Optional ("invoice.macaroon" by default).
	MacaroonFile string
}

// DefaultLNDoptions provides default values for LNDoptions.
var DefaultLNDoptions = LNDoptions{
	Address:      "localhost:10009",
	CertFile:     "tls.cert",
	MacaroonFile: "invoice.macaroon",
}

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

// NewGinMiddleware returns a Gin middleware in the form of a gin.HandlerFunc.
func NewGinMiddleware(invoiceOptions InvoiceOptions, lndOptions LNDoptions, storageClient StorageClient) gin.HandlerFunc {
	invoiceOptions, lndOptions = assignDefaultValues(invoiceOptions, lndOptions)
	client, err := ln.NewLNDclient(lndOptions.Address, lndOptions.CertFile, lndOptions.MacaroonFile)
	if err != nil {
		panic(err)
	}
	return func(ctx *gin.Context) {
		// Check if the request contains a header with the preimage that we need to check if the requester paid
		preimage := ctx.GetHeader("x-preimage")
		if preimage == "" {
			// Generate the invoice
			invoice, err := client.GenerateInvoice(invoiceOptions.Price, invoiceOptions.Memo)
			if err != nil {
				errorMsg := fmt.Sprintf("Couldn't generate invoice: %+v", err)
				log.Println(errorMsg)
				http.Error(ctx.Writer, errorMsg, http.StatusInternalServerError)
				ctx.Abort()
			} else {
				stdOutLogger.Printf("Sending invoice in response: %v", invoice)
				ctx.Header("Content-Type", "application/vnd.lightning.bolt11")
				ctx.Status(http.StatusPaymentRequired)
				// The actual invoice goes into the body
				ctx.Writer.Write([]byte(invoice))
				ctx.Abort()
			}
		} else {
			// Check if the provided preimage belongs to a settled API payment invoice and that it wasn't already used and store used preimages
			ok, err := handlePreimage(preimage, storageClient, client)
			if err != nil {
				errorMsg := fmt.Sprintf("An error occured during checking the preimage: %+v", err)
				log.Printf("%v\n", errorMsg)
				http.Error(ctx.Writer, errorMsg, http.StatusInternalServerError)
				ctx.Abort()
			} else {
				if !ok {
					log.Printf("The provided preimage is invalid: %v\n", preimage)
					http.Error(ctx.Writer, "The provided preimage is invalid", http.StatusBadRequest)
					ctx.Abort()
				} else {
					preimageHash, err := ln.HashPreimage(preimage)
					if err == nil {
						stdOutLogger.Printf("The provided preimage is valid. Continuing to the next handler. Preimage hash: %v\n", preimageHash)
					}
					ctx.Next()
				}
			}
		}
	}
}

// NewEchoMiddleware returns an Echo middleware in the form of an echo.MiddlewareFunc.
func NewEchoMiddleware(invoiceOptions InvoiceOptions, lndOptions LNDoptions, storageClient StorageClient, skipper middleware.Skipper) echo.MiddlewareFunc {
	invoiceOptions, lndOptions = assignDefaultValues(invoiceOptions, lndOptions)
	client, err := ln.NewLNDclient(lndOptions.Address, lndOptions.CertFile, lndOptions.MacaroonFile)
	if err != nil {
		panic(err)
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		if skipper == nil {
			skipper = middleware.DefaultSkipper
		}
		return func(ctx echo.Context) error {
			if skipper(ctx) {
				return next(ctx)
			}
			// Check if the request contains a header with the preimage that we need to check if the requester paid
			preimage := ctx.Request().Header.Get("x-preimage")
			if preimage == "" {
				// Generate the invoice
				invoice, err := client.GenerateInvoice(invoiceOptions.Price, invoiceOptions.Memo)
				if err != nil {
					errorMsg := fmt.Sprintf("Couldn't generate invoice: %+v", err)
					log.Println(errorMsg)
					return &echo.HTTPError{
						Code:     http.StatusInternalServerError,
						Message:  errorMsg,
						Internal: err,
					}
				} else {
					stdOutLogger.Printf("Sending invoice in response: %v", invoice)
					ctx.Response().Header().Set("Content-Type", "application/vnd.lightning.bolt11")
					ctx.Response().Status = http.StatusPaymentRequired
					// The actual invoice goes into the body
					ctx.Response().Write([]byte(invoice))
					return &echo.HTTPError{
						Code:    http.StatusPaymentRequired,
						Message: invoice,
					}
				}
			} else {
				// Check if the provided preimage belongs to a settled API payment invoice and that it wasn't already used and store used preimages
				ok, err := handlePreimage(preimage, storageClient, client)
				if err != nil {
					errorMsg := fmt.Sprintf("An error occured during checking the preimage: %+v", err)
					log.Printf("%v\n", errorMsg)
					return &echo.HTTPError{
						Code:     http.StatusInternalServerError,
						Message:  errorMsg,
						Internal: err,
					}
				} else {
					if !ok {
						errorMsg := "The provided preimage is invalid"
						log.Printf("%v: %v\n", errorMsg, preimage)
						return &echo.HTTPError{
							Code:     http.StatusBadRequest,
							Message:  errorMsg,
							Internal: err,
						}
					} else {
						preimageHash, err := ln.HashPreimage(preimage)
						if err == nil {
							stdOutLogger.Printf("The provided preimage is valid. Continuing to the next HandlerFunc. Preimage hash: %v\n", preimageHash)
						}
					}
				}
			}
			return next(ctx)
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

// LNclient is an abstraction of a client that connects to a Lightning Network node implementation (like lnd, c-lightning and eclair)
// and provides the methods required by the paywall.
type LNclient interface {
	GenerateInvoice(int64, string) (string, error)
	CheckInvoice(string) (bool, error)
}

// handlePreimage does four things:
// 1) Checks if the preimage was already used as a payment proof before.
// 2) Checks if the preimage corresponds to an existing invoice on the connected lnd.
// 3) Checks if the corresponding invoice was settled.
// 4) Store the preimage to the storage for future checks.
// Returns false and an empty error if the preimage was already used or if the corresponding invoice isn't settled.
// Returns true and an empty error if everythings's fine.
func handlePreimage(preimage string, storageClient StorageClient, lndClient LNclient) (bool, error) {
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
	settled, err := lndClient.CheckInvoice(preimage)
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

func assignDefaultValues(invoiceOptions InvoiceOptions, lndOptions LNDoptions) (InvoiceOptions, LNDoptions) {
	// InvoiceOptions
	if invoiceOptions.Price <= 0 {
		invoiceOptions.Price = DefaultInvoiceOptions.Price
	}
	// Empty Memo is okay.

	// LNDoptions
	if lndOptions.Address == "" {
		lndOptions.Address = DefaultLNDoptions.Address
	}
	if lndOptions.CertFile == "" {
		lndOptions.CertFile = DefaultLNDoptions.CertFile
	}
	if lndOptions.MacaroonFile == "" {
		lndOptions.MacaroonFile = DefaultLNDoptions.MacaroonFile
	}

	return invoiceOptions, lndOptions
}
