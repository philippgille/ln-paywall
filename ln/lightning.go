package ln

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/philippgille/ln-paywall/lnrpc"
)

// GenerateInvoice generates an invoice with the given amount.
// For doing so, a gRPC connection to the given address is established, using the given cert and macaroon files.
func GenerateInvoice(amount int64, memo string, address string, certFile string, macaroonFile string) (string, error) {
	// Create the client
	c, ctx, conn, err := NewLightningClient(address, certFile, macaroonFile)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Create the request and send it
	if memo == "" {
		memo = "API call"
	}
	invoice := lnrpc.Invoice{
		Memo:  memo,
		Value: amount,
	}
	log.Println("Creating invoice for a new API request")
	res, err := c.AddInvoice(ctx, &invoice)
	if err != nil {
		return "", err
	}

	return res.GetPaymentRequest(), nil
}

// CheckPreimage takes a Base64 encoded preimage and checks if it's a valid preimage for an API payment.
// For doing so, a gRPC connection to the given address is established, using the given cert and macaroon files.
func CheckPreimage(preimage string, address string, certFile string, macaroonFile string) (bool, error) {
	// Hash the preimage so we can get the invoice that belongs to it to check if it's settled

	decodedPreimage, err := base64.StdEncoding.DecodeString(preimage)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256([]byte(decodedPreimage))
	hashSlice := hash[:]

	// Get the invoice for that hash

	// Create the client
	c, ctx, conn, err := NewLightningClient(address, certFile, macaroonFile)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	// Create the request and send it
	paymentHash := lnrpc.PaymentHash{
		RHash: hashSlice,
		// Hex encoded, must be exactly 32 byte
		RHashStr: hex.EncodeToString(hashSlice),
	}
	encodedHash := base64.StdEncoding.EncodeToString(hashSlice)
	log.Printf("Checking invoice for hash %v\n", encodedHash)
	invoice, err := c.LookupInvoice(ctx, &paymentHash)
	if err != nil {
		return false, err
	}

	// Perform checks on the invoice

	// Check if invoice was settled
	if !invoice.GetSettled() {
		return false, nil
	}

	// Check if it was already used before
	// TODO: implement

	return true, nil
}

// NewLightningClient creates a new gRPC connection to the given address
// and creates a new client using the given cert and macaroon files.
// One of the return values is the gRPC connection, which the calling function MUST close.
func NewLightningClient(address string, certFile string, macaroonFile string) (lnrpc.LightningClient, context.Context, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	creds, err := credentials.NewClientTLSFromFile(certFile, "")
	if err != nil {
		return nil, nil, nil, err
	}
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, nil, nil, err
	}
	c := lnrpc.NewLightningClient(conn)

	// Add the macaroon to the outgoing context

	macaroon, err := ioutil.ReadFile(macaroonFile)
	if err != nil {
		return nil, nil, nil, err
	}
	// Value must be the hex representation of the file content
	macaroonHex := fmt.Sprintf("%X", string(macaroon))
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "macaroon", macaroonHex)

	return c, ctx, conn, nil
}

// HashPreimage hashes the Base64 preimage and encodes the hash in Base64.
// It's the same format that's being shown by lncli listinvoices (preimage as well as hash).
func HashPreimage(preimage string) (string, error) {
	decodedPreimage, err := base64.StdEncoding.DecodeString(preimage)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(decodedPreimage))
	hashSlice := hash[:]
	encodedHash := base64.StdEncoding.EncodeToString(hashSlice)
	return encodedHash, nil
}
