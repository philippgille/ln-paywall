package pay

import (
	"net/http"
)

// NewMiddleware returns a function which you can use within an http.HandlerFunc chain.
// The parameter is the amount of satoshis you want to have paid for one API call.
func NewMiddleware(amount int64) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Check if the request contains a header with the token that we need to check if the requester paid
			token := r.Header.Get("x-token")
			if token == "" {
				// Note: w.Header().Set(...) must be called before w.WriteHeader(...)!
				w.Header().Set("Content-Type", "application/vnd.lightning.bolt11")
				w.WriteHeader(http.StatusPaymentRequired)
				// The actual invoice goes into the body
				invoice := generateInvoice(amount)
				w.Write([]byte(invoice))
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

func generateInvoice(amount int64) string {
	return "not implemented" // TODO: implement
}

func checkToken(token string) bool {
	return false // TODO: implement
}
