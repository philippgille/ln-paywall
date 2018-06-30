package pay

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/philippgille/go-paypercall/lnrpc"
)

// NewMiddleware returns a function which you can use within an http.HandlerFunc chain.
// The amount parameter is the amount of satoshis you want to have paid for one API call.
// The address parameter is the address of your LND node, including the port.
// The certFile parameter is the path to the "tls.cert" file that your LND node uses.
// The macaroonFile parameter is the path to the "admin.macaroon" file that your LND node uses.
func NewMiddleware(amount int64, address string, certFile string, macaroonFile string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Check if the request contains a header with the token that we need to check if the requester paid
			token := r.Header.Get("x-token")
			if token == "" {
				// Generate the invoice
				invoice, err := generateInvoice(amount, address, certFile, macaroonFile)
				if err != nil {
					errorMsg := fmt.Sprintf("Couldn't generate invoice: %+v", err)
					http.Error(w, errorMsg, http.StatusBadRequest)
				} else {
					// Note: w.Header().Set(...) must be called before w.WriteHeader(...)!
					w.Header().Set("Content-Type", "application/vnd.lightning.bolt11")
					w.WriteHeader(http.StatusPaymentRequired)
					// The actual invoice goes into the body
					w.Write([]byte(invoice))
				}
			} else {
				// Get the token and check if the payment is OK
				ok := checkToken(token)
				if !ok {
					http.Error(w, "Provided token is invalid", http.StatusBadRequest)
				} else {
					next.ServeHTTP(w, r)
				}
			}
		}
	}
}

func generateInvoice(amount int64, address string, certFile string, macaroonFile string) (string, error) {
	// Set up a connection to the server.
	creds, err := credentials.NewClientTLSFromFile(certFile, "")
	if err != nil {
		return "", err
	}
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return "", err
	}
	defer conn.Close()
	c := lnrpc.NewLightningClient(conn)

	// Add the macaroon to the outgoing context

	macaroon, err := ioutil.ReadFile(macaroonFile)
	if err != nil {
		return "", err
	}
	// Value must be the hex representation of the file content
	macaroonHex := fmt.Sprintf("%X", string(macaroon))
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "macaroon", macaroonHex)

	// Create the request and send it

	invoice := lnrpc.Invoice{
		Memo:  "API call payment",
		Value: amount,
	}
	res, err := c.AddInvoice(ctx, &invoice)
	if err != nil {
		return "", err
	}

	return res.GetPaymentRequest(), nil
}

func checkToken(token string) bool {
	return false // TODO: implement
}
