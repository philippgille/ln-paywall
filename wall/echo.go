package wall

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
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
			fa := echoAbstraction{
				ctx:         ctx,
				nextHandler: next,
			}
			return commonHandler(fa, invoiceOptions, lnClient, storageClient)
		}
	}
}

type echoAbstraction struct {
	ctx         echo.Context
	nextHandler echo.HandlerFunc
}

func (fa echoAbstraction) getPreimageFromHeader() string {
	return fa.ctx.Request().Header.Get("x-preimage")
}

func (fa echoAbstraction) respondWithError(err error, errorMsg string, statusCode int) {
	fa.ctx.String(statusCode, errorMsg)
}

func (fa echoAbstraction) getHTTPrequest() *http.Request {
	return fa.ctx.Request()
}

func (fa echoAbstraction) respondWithInvoice(headers map[string]string, statusCode int, body []byte) {
	for k, v := range headers {
		fa.ctx.Response().Header().Set(k, v)
	}
	fa.ctx.String(statusCode, string(body))
}

func (fa echoAbstraction) next() error {
	return fa.nextHandler(fa.ctx)
}
