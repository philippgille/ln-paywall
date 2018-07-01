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

// NewHandlerFuncMiddleware returns a function which you can use within an http.HandlerFunc chain.
// The amount parameter is the amount of satoshis you want to have paid for one API call.
// The address parameter is the address of your LND node, including the port.
// The certFile parameter is the path to the "tls.cert" file that your LND node uses.
// The macaroonFile parameter is the path to the "admin.macaroon" file that your LND node uses.
func NewHandlerFuncMiddleware(amount int64, address string, certFile string, macaroonFile string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return createHandlerFunc(amount, address, certFile, macaroonFile, next)
	}
}

// NewHandlerMiddleware returns a function which you can use within an http.Handler chain.
// The amount parameter is the amount of satoshis you want to have paid for one API call.
// The address parameter is the address of your LND node, including the port.
// The certFile parameter is the path to the "tls.cert" file that your LND node uses.
// The macaroonFile parameter is the path to the "admin.macaroon" file that your LND node uses.
func NewHandlerMiddleware(amount int64, address string, certFile string, macaroonFile string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(createHandlerFunc(amount, address, certFile, macaroonFile, next.ServeHTTP))
	}
}

func createHandlerFunc(amount int64, address string, certFile string, macaroonFile string, next http.HandlerFunc) func(w http.ResponseWriter, r *http.Request) {
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
				stdOutLogger.Printf("Sending invoice in response: %v", invoice)
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
// The amount parameter is the amount of satoshis you want to have paid for one API call.
// The address parameter is the address of your LND node, including the port.
// The certFile parameter is the path to the "tls.cert" file that your LND node uses.
// The macaroonFile parameter is the path to the "admin.macaroon" file that your LND node uses.
func NewGinMiddleware(amount int64, address string, certFile string, macaroonFile string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Check if the request contains a header with the preimage that we need to check if the requester paid
		preimage := ctx.GetHeader("x-preimage")
		if preimage == "" {
			// Generate the invoice
			invoice, err := ln.GenerateInvoice(amount, address, certFile, macaroonFile)
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
			// Check if the provided preimage belongs to a settled API payment invoice and that it wasn't already used
			ok, err := ln.CheckPreimage(preimage, address, certFile, macaroonFile)
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
