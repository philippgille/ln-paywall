package wall

import (
	"net/http"
)

// NewHandlerFuncMiddleware returns a function which you can use within an http.HandlerFunc chain.
func NewHandlerFuncMiddleware(invoiceOptions InvoiceOptions, lnClient LNclient, storageClient StorageClient) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return createHandlerFunc(invoiceOptions, lnClient, storageClient, next)
	}
}

// NewHandlerMiddleware returns a function which you can use within an http.Handler chain.
func NewHandlerMiddleware(invoiceOptions InvoiceOptions, lnClient LNclient, storageClient StorageClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(createHandlerFunc(invoiceOptions, lnClient, storageClient, next.ServeHTTP))
	}
}

func createHandlerFunc(invoiceOptions InvoiceOptions, lnClient LNclient, storageClient StorageClient, next http.HandlerFunc) func(w http.ResponseWriter, r *http.Request) {
	invoiceOptions = assignDefaultValues(invoiceOptions)
	return func(w http.ResponseWriter, r *http.Request) {
		fa := stdlibHTTP{
			w:           w,
			r:           r,
			nextHandler: next,
		}
		commonHandler(fa, invoiceOptions, lnClient, storageClient)
	}
}

type stdlibHTTP struct {
	w           http.ResponseWriter
	r           *http.Request
	nextHandler http.HandlerFunc
}

func (fa stdlibHTTP) getPreimageFromHeader() string {
	return fa.r.Header.Get("x-preimage")
}

func (fa stdlibHTTP) respondWithError(err error, errorMsg string, statusCode int) {
	http.Error(fa.w, errorMsg, statusCode)
}

func (fa stdlibHTTP) getHTTPrequest() *http.Request {
	return fa.r
}

func (fa stdlibHTTP) respondWithInvoice(headers map[string]string, statusCode int, body []byte) {
	// Note: w.Header().Set(...) must be called before w.WriteHeader(...)!
	for k, v := range headers {
		fa.w.Header().Set(k, v)
	}
	// Status code
	fa.w.WriteHeader(statusCode)
	// The actual invoice goes into the body
	fa.w.Write(body)
}

func (fa stdlibHTTP) next() error {
	fa.nextHandler.ServeHTTP(fa.w, fa.r)
	return nil
}
