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
	// The actual invoice string required by the payer in Bech32 encoding,
	// see https://github.com/lightningnetwork/lightning-rfc/blob/master/11-payment-encoding.md
	PaymentRequest string
	// A.k.a. preimage hash.
	// Could be extracted from the PaymentRequest, but that would require additional
	// dependencies during build time and additional computation during runtime,
	// while all Lightning Node implementation clients already return the value directly
	// when generating a new invoice.
	PaymentHash string
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
