package ln

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
)

// stdOutLogger logs to stdout, while the default log package loggers log to stderr.
var stdOutLogger = log.New(os.Stdout, "", log.LstdFlags)

// Invoice is a Lightning Network invoice and contains the typical invoice string and the payment hash.
type Invoice struct {
	// The unique identifier for the invoice in the LN node.
	// The value depends on the LN node implementation.
	//
	// For example, lnd uses the payment hash (a.k.a. preimage hash) as ID.
	// It doesn't use this term ("ID"), but when fetching a single invoice via RPC,
	// the payment hash is used as identifier.
	// Also, Lightning Lab's (creators of lnd) desktop app "Lightning" explicitly shows
	// the payment hash in a field with the title "invoice ID" in the Lightning transaction overview.
	//
	// But Lightning Charge on the other hand generates its own timestamp-based ID for each invoice.
	// They explicitly call that value "invoice ID" and they also require it when fetching a single invoice
	// via Lightning Charge's RESTful API.
	ImplDepID string
	// A.k.a. preimage hash. Hex encoded.
	// Could be extracted from the PaymentRequest, but that would require additional
	// dependencies during build time and additional computation during runtime,
	// while all Lightning Node implementation clients already return the value directly
	// when generating an invoice.
	PaymentHash string
	// The actual invoice string required by the payer in Bech32 encoding,
	// see https://github.com/lightningnetwork/lightning-rfc/blob/master/11-payment-encoding.md
	PaymentRequest string
}

// HashPreimage turns a hex encoded preimage into a hex encoded preimage hash.
// It's the same format that's being used by "lncli listpayments", Eclair on Android and bolt11 payment request decoders like https://lndecode.com.
// Only "lncli listinvoices" uses Base64.
func HashPreimage(preimageHex string) (string, error) {
	// Decode from hex, hash, encode to hex
	preimage, err := hex.DecodeString(preimageHex)
	if err != nil {
		return "", err
	}
	hashByteArray := sha256.Sum256(preimage)
	preimageHashHex := hex.EncodeToString(hashByteArray[:])
	return preimageHashHex, nil
}
