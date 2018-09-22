package wall

import (
	"encoding/hex"
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
	// Set stores the given invoiceMetaData for the given invoice ID.
	Set(string, interface{}) error
	// Get retrieves the invoiceMetaData for the given invoice ID
	// and populates the fields of the object that the passed pointer
	// points to with the values of the retrieved object's values.
	// If no object is found it returns (false, nil).
	Get(string, interface{}) (bool, error)
}

// LNclient is an abstraction of a client that connects to a Lightning Network node implementation (like lnd, c-lightning and eclair)
// and provides the methods required by the paywall.
type LNclient interface {
	GenerateInvoice(int64, string) (ln.Invoice, error)
	CheckInvoice(string) (bool, error)
}

// invoiceMetaData is data that's required to prevent clients from cheating
// (e.g. have multiple requests executed while having paid only once,
// or requesting an invoice for a cheap endpoint and using the payment proof for an expensive one).
// The type itself is not exported, but the fields have to be (for (un-)marshaling).
type invoiceMetaData struct {
	Method string
	Path   string
	Used   bool
}

// handlePreimage does the following:
// 1) Validate the preimage format (encoding, length)
// 2) Check if the invoice ID exists in the storage
// 3) Check if the current HTTP verb and URL path match the ones used for creating the invoice
// 4) Check if the invoice ID was already used in a previous request
// 5) Check if the invoice was settled
// 6) Mark the invoice ID as used, so it can't be used in future requests
// Note: The invoice ID (a.k.a. payment hash, a.k.a. preimage hash) can be calculated from the preimage.
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

	// Calculate invoice ID from preimage.
	// Ignore error because we already validated the preimage format.
	invoiceID, _ := ln.HashPreimage(preimage)

	// Retrieve invoice metadata from storage
	metaData := new(invoiceMetaData)
	found, err := storageClient.Get(invoiceID, metaData)
	if err != nil {
		return "", err
	}

	// Execute all checks that we can do locally.

	// 2. Check if the invoice ID exists in the storage
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
	// 4) Check if the invoice ID was already used in a previous request
	if metaData.Used {
		return "You already sent a request with the same preimage. You have to pay a new invoice for and include the corresponding preimage in each request.", nil
	}

	// 5) Check if the invoice was settled
	settled, err := lnClient.CheckInvoice(preimage)
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

	// 6) Mark the invoice ID as used, so it can't be used in future requests
	metaData.Used = true
	err = storageClient.Set(invoiceID, *metaData)
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
