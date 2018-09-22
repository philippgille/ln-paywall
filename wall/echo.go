package wall

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/philippgille/ln-paywall/ln"
)

// NewEchoMiddleware returns an Echo middleware in the form of an echo.MiddlewareFunc.
func NewEchoMiddleware(invoiceOptions InvoiceOptions, lnClient LNclient, storageClient StorageClient, skipper middleware.Skipper) echo.MiddlewareFunc {
	invoiceOptions = assignDefaultValues(invoiceOptions)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		if skipper == nil {
			skipper = middleware.DefaultSkipper
		}
		return func(ctx echo.Context) error {
			if skipper(ctx) {
				return next(ctx)
			}
			// Check if the request contains a header with the preimage that we need to check if the requester paid
			preimageHex := ctx.Request().Header.Get("x-preimage")
			if preimageHex == "" {
				// Generate the invoice
				invoice, err := lnClient.GenerateInvoice(invoiceOptions.Price, invoiceOptions.Memo)
				if err != nil {
					errorMsg := fmt.Sprintf("Couldn't generate invoice: %+v", err)
					log.Println(errorMsg)
					return &echo.HTTPError{
						Code:     http.StatusInternalServerError,
						Message:  errorMsg,
						Internal: err,
					}
				}

				// Cache the invoice metadata
				metadata := invoiceMetaData{
					Method: ctx.Request().Method,
					Path:   ctx.Request().URL.Path,
				}
				storageClient.Set(invoice.PaymentHash, metadata)

				stdOutLogger.Printf("Sending invoice in response: %v", invoice)
				ctx.Response().Header().Set("Content-Type", "application/vnd.lightning.bolt11")
				ctx.Response().Status = http.StatusPaymentRequired
				// The actual invoice goes into the body
				ctx.Response().Write([]byte(invoice.PaymentRequest))
				return &echo.HTTPError{
					Code:    http.StatusPaymentRequired,
					Message: invoice,
				}
			}
			// Check if the provided preimage belongs to a settled API payment invoice and that it wasn't already used. Also store used preimages.
			invalidPreimageMsg, err := handlePreimage(ctx.Request(), storageClient, lnClient)
			if err != nil {
				errorMsg := fmt.Sprintf("An error occurred during checking the preimage: %+v", err)
				log.Printf("%v\n", errorMsg)
				return &echo.HTTPError{
					Code:     http.StatusInternalServerError,
					Message:  errorMsg,
					Internal: err,
				}
			} else if invalidPreimageMsg != "" {
				log.Printf("%v: %v\n", invalidPreimageMsg, preimageHex)
				return &echo.HTTPError{
					Code:     http.StatusBadRequest,
					Message:  invalidPreimageMsg,
					Internal: err,
				}
			} else {
				preimageHash, err := ln.HashPreimage(preimageHex)
				if err == nil {
					stdOutLogger.Printf("The provided preimage is valid. Continuing to the next HandlerFunc. Preimage hash: %v\n", preimageHash)
				}
			}
			return next(ctx)
		}
	}
}
