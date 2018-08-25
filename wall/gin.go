package wall

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/philippgille/ln-paywall/ln"
)

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
