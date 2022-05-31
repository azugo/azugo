# azugo

Opinionated GoLang web framework for microservices based on FastHTTP.

### Features

* HTTP web server [valyala/fasthttp](https://github.com/valyala/fasthttp)
* HTTP/2 support [dgrr/http2](https://github.com/dgrr/http2)
* Web socket support [dgrr/websocket](https://github.com/dgrr/websocket)
* Structured logger [go.uber.org/zap](https://github.com/uber-go/zap)
* JSON serialization [goccy/go-json](https://github.com/goccy/go-json)
* Data structure validation using [go-playground/validator](https://github.com/go-playground/validator)
* Built-in web app testing framework

### Special Environment variables used by the Azugo framework

* `ENVIRONMENT` - An App environment setting (allowed values are `Development`, `Staging` and `Production`).
* `BASE_PATH` - Base path for an App if it's deployed in subdirectory.
* `SERVER_URL` - An server URL to listen on.

### Special thanks to

* Router largely based on [fasthttp/router](https://github.com/fasthttp/router)
