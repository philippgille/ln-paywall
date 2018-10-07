package wall

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"

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

// StorageClient is an abstraction for different storage client implementations.
// A storage client must be able to store and retrieve invoiceMetaData objects.
type StorageClient interface {
	// Set stores the given invoiceMetaData for the given preimage hash.
	Set(string, interface{}) error
	// Get retrieves the invoiceMetaData for the given preimage hash
	// and populates the fields of the object that the passed pointer
	// points to with the values of the retrieved object's values.
	// If no object is found it returns (false, nil).
	Get(string, interface{}) (bool, error)
}

// LNclient is an abstraction of a client that connects to a Lightning Network node implementation (like lnd, c-lightning and eclair)
// and provides the methods required by the paywall.
type LNclient interface {
	// GenerateInvoice generates a new invoice based on the price in Satoshis and with the given memo.
	GenerateInvoice(int64, string) (ln.Invoice, error)
	// CheckInvoice checks if the invoice was settled, given an LN node implementation dependent ID.
	// For example lnd uses the payment hash a.k.a. preimage hash as ID, while Lightning Charge
	// uses a randomly generated string as ID.
	CheckInvoice(string) (bool, error)
}

// invoiceMetaData is data that's required to prevent clients from cheating
// (e.g. have multiple requests executed while having paid only once,
// or requesting an invoice for a cheap endpoint and using the payment proof for an expensive one).
// The type itself is not exported, but the fields have to be (for (un-)marshaling).
type invoiceMetaData struct {
	// The unique identifier for the invoice in the LN node.
	// This is NOT the ID that's used for storing the metadata in the storage.
	// Instead, it's the ID used to retrieve info about an invoice from the LN node.
	// The different implementations use different values as ID, for example
	// lnd uses the payment hash a.k.a. preimage hash as ID, while Lightning Charge
	// uses its own randomly generated string as ID.
	//
	// The ID (or rather *key*) used for storing the metadata in the storage
	// is the payment hash of the invoice, because the client sends the preimage
	// (or in the future also its hash) in the final request and we must be able
	// to look up the metadata with that key.
	ImplDepID string
	Method    string
	Path      string
	Used      bool
}

type frameworkAbstraction interface {
	// getPreimageFromHeader returns the content of the "X-Preimage" header.
	getPreimageFromHeader() string
	// respondWithError sends a response with the given message and status code.
	respondWithError(error, string, int)
	// getHTTPrequest returns a pointer to the current http.Request.
	getHTTPrequest() *http.Request
	// respondWithInvoice sends a response with the given headers, status code and invoice string.
	respondWithInvoice(map[string]string, int, []byte)
	// next moves to the next handler, which might be another middleware or the actual request handler.
	// This method is only called when all previous operations were successful (e.g. the invoice was paid properly).
	// An error only needs to be returned if the specific web framework requires middlewares to return one,
	// like Echo does for example.
	next() error
}

func commonHandler(fa frameworkAbstraction, invoiceOptions InvoiceOptions, lnClient LNclient, storageClient StorageClient) error {
	// Check if the request contains a header with the preimage that we need to check if the requester paid
	preimageHex := fa.getPreimageFromHeader()
	if preimageHex == "" {
		// Generate the invoice
		invoice, err := lnClient.GenerateInvoice(invoiceOptions.Price, invoiceOptions.Memo)
		if err != nil {
			errorMsg := fmt.Sprintf("Couldn't generate invoice: %+v", err)
			log.Println(errorMsg)
			fa.respondWithError(err, errorMsg, http.StatusInternalServerError)
		} else {
			// Cache the invoice metadata
			metadata := invoiceMetaData{
				ImplDepID: invoice.ImplDepID,
				Method:    fa.getHTTPrequest().Method,
				Path:      fa.getHTTPrequest().URL.Path,
			}
			storageClient.Set(invoice.PaymentHash, metadata)

			// Respond with the invoice
			stdOutLogger.Printf("Sending invoice in response: %v", invoice.PaymentRequest)
			headers := make(map[string]string)
			headers["Content-Type"] = "application/vnd.lightning.bolt11"
			fa.respondWithInvoice(headers, http.StatusPaymentRequired, []byte(invoice.PaymentRequest))
		}
	} else {
		// Check if the provided preimage belongs to a settled API payment invoice and that it wasn't already used. Also store used preimages.
		invalidPreimageMsg, err := handlePreimage(fa.getHTTPrequest(), storageClient, lnClient)
		if err != nil {
			errorMsg := fmt.Sprintf("An error occurred during checking the preimage: %+v", err)
			log.Printf("%v\n", errorMsg)
			fa.respondWithError(err, errorMsg, http.StatusInternalServerError)
		} else if invalidPreimageMsg != "" {
			log.Printf("%v: %v\n", invalidPreimageMsg, preimageHex)
			fa.respondWithError(nil, invalidPreimageMsg, http.StatusBadRequest)
		} else {
			// The preimage was valid (has a corresponding + settled invoice, wasn't used before etc.). Continue to next handler.
			preimageHash, err := ln.HashPreimage(preimageHex)
			if err == nil {
				stdOutLogger.Printf("The provided preimage is valid. Continuing to the next handler. Preimage hash: %v\n", preimageHash)
			}
			err = fa.next()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// handlePreimage does the following:
// 1) Validate the preimage format (encoding, length)
// 2) Check if the invoice metadata exists in the storage
// 3) Check if the current HTTP verb and URL path match the ones used for creating the invoice
// 4) Check if the payment hash was already used in a previous request
// 5) Check if the invoice was settled
// 6) Mark the invoice metadata as used, so it can't be used in future requests
// Note: The payment hash (a.k.a. preimage hash) can be calculated from the preimage.
//
// Returns a string and an error.
// The string contains detailed info about the result in case the preimage is invalid
// (bad encoding, HTTP verb doesn't match, already used etc., generally a client-side error).
// The error is only non-nil if a server-side error occurred during the check (like the LN node can't be reached).
// The preimage is only valid if the string is empty and the error is nil.
func handlePreimage(req *http.Request, storageClient StorageClient, lnClient LNclient) (string, error) {
	// 1) Validate the preimage format (encoding, length)
	preimage := req.Header.Get("X-Preimage")
	errString := validatePreimageFormat(preimage)
	if errString != "" {
		return errString, nil
	}

	// Calculate preimage hash (a.k.a. payment hash) from preimage.
	// Ignore error because we already validated the preimage format.
	preimageHash, _ := ln.HashPreimage(preimage)

	// Retrieve invoice metadata from storage
	metaData := new(invoiceMetaData)
	found, err := storageClient.Get(preimageHash, metaData)
	if err != nil {
		return "", err
	}

	// Execute all checks that we can do locally.

	// 2. Check if the preimage hash exists in the storage
	if !found {
		return "You seem to have sent an invalid preimage or one that doesn't correspond to an invoice that was issued for an initial request", nil
	}
	// 3) Check if the current HTTP verb and URL path match the ones used for creating the invoice
	if req.Method != metaData.Method {
		return "Your invoice was created for a " + metaData.Method + " request, but you're sending a " + req.Method + " request", nil
	}
	if req.URL.Path != metaData.Path {
		return "Your invoice was created for the path \"" + metaData.Path + "\", but you're sending a request to \"" + req.URL.Path + "\"", nil
	}
	// 4) Check if the preimage hash was already used in a previous request
	if metaData.Used {
		return "You already sent a request with the same preimage. You have to pay a new invoice for and include the corresponding preimage in each request.", nil
	}

	// 5) Check if the invoice was settled
	settled, err := lnClient.CheckInvoice(metaData.ImplDepID)
	if err != nil {
		// Returning a non-nil error leads to an "internal server error", but in some cases it's a "bad request".
		// Handle those cases here.
		// TODO: Checks should be done in a more robust and elegant way
		if reflect.TypeOf(err).Name() == "InvalidByteError" ||
			err == hex.ErrLength {
			return "The provided preimage isn't properly hex encoded", nil
		} else if strings.Contains(err.Error(), "unable to locate invoice") {
			return "No corresponding invoice was found for the provided preimage", nil
		} else {
			return "", err
		}
	}
	if !settled {
		return "You somehow obtained the preimage of the invoice, but the invoice is not settled yet", nil
	}

	// 6) Mark the invoice as used, so it can't be used in future requests
	metaData.Used = true
	err = storageClient.Set(preimageHash, *metaData)
	if err != nil {
		return "", err
	}

	return "", nil
}

func validatePreimageFormat(preimageHex string) string {
	if len(preimageHex) != 64 {
		return "The provided preimage isn't properly formatted"
	}
	_, err := hex.DecodeString(preimageHex)
	if err != nil {
		// Either err == hex.ErrLength or err == hex.InvalidByteError.
		return "The provided preimage isn't properly hex encoded"
	}
	return ""
}

func assignDefaultValues(invoiceOptions InvoiceOptions) InvoiceOptions {
	// InvoiceOptions
	if invoiceOptions.Price <= 0 {
		invoiceOptions.Price = DefaultInvoiceOptions.Price
	}
	// Empty Memo is okay.

	return invoiceOptions
}
