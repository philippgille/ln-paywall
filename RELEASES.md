Releases
========

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

vNext
-----

v0.1.0 (2018-07-02)
-------------------

Initial release after working on the project during the "Chainhack 3" [Blockchain hackathon](https://blockchain-hackathon.com/).

- Go middlewares for:
    - [net/http](https://golang.org/pkg/net/http/) `HandlerFunc`
    - [net/http](https://golang.org/pkg/net/http/) `Handler`
        - Compatible with routers like [gorilla/mux](https://github.com/gorilla/mux) and [chi](https://github.com/go-chi/chi)
    - [Gin](https://github.com/gin-gonic/gin)
- Supported Lightning Network node:
    - [lnd](https://github.com/lightningnetwork/lnd)
- Supported storage:
    - [Redis](https://redis.io/)
