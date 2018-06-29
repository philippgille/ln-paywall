go-paypercall
=============

Go middleware for monetizing your API on a pay-per-call basis with Bitcoin and Lightning ⚡️

We're first focusing on `net/http`-compatible middleware and later adding middleware for [Gin](https://github.com/gin-gonic/gin), [Echo](https://github.com/labstack/echo) and other popular Go web frameworks.

An API gateway is on the roadmap as well, which you can use to monetize your API that's written in *any* language, so no need to use Go.

Prior Art
---------

- [https://github.com/ElementsProject/paypercall](https://github.com/ElementsProject/paypercall)
    - Middleware for the JavaScript web framework [Express](https://expressjs.com/)
    - Reverse proxy
    - Payment: Lightning Network
- [https://github.com/interledgerjs/koa-web-monetization](https://github.com/interledgerjs/koa-web-monetization)
    - Middleware for the JavaScript web framework [Koa](https://koajs.com/)
    - Payment: Interledger
- [https://moonbanking.com/api](https://moonbanking.com/api)
    - API that *uses* a similar functionality, not *providing* it
