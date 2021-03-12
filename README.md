# logger

[![Build Status](https://travis-ci.org/gin-contrib/logger.svg?branch=master)](https://travis-ci.org/gin-contrib/logger)
[![codecov](https://codecov.io/gh/gin-contrib/logger/branch/master/graph/badge.svg)](https://codecov.io/gh/gin-contrib/logger)
[![Go Report Card](https://goreportcard.com/badge/github.com/gin-contrib/logger)](https://goreportcard.com/report/github.com/gin-contrib/logger)
[![GoDoc](https://godoc.org/github.com/gin-contrib/logger?status.svg)](https://godoc.org/github.com/gin-contrib/logger)
[![Join the chat at https://gitter.im/gin-gonic/gin](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/gin-gonic/gin)

Gin middleware/handler to logger url path using [rs/zerolog](https://github.com/rs/zerolog).

## Example

```go
package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	rxURL = regexp.MustCompile(`^/regexp\d*`)
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if gin.IsDebugging() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Logger = log.Output(
		zerolog.ConsoleWriter{
			Out:     os.Stderr,
			NoColor: false,
		},
	)

	r := gin.New()

	// Add a logger middleware, which:
	//   - Logs all requests, like a combined access and error log.
	//   - Logs to stdout.
	r.Use(logger.SetLogger())

	// Custom logger
	subLog := zerolog.New(os.Stdout).With().
		Str("foo", "bar").
		Logger()

	r.Use(logger.SetLogger(logger.Config{
		Logger:         &subLog,
		UTC:            true,
		SkipPath:       []string{"/skip"},
		SkipPathRegexp: rxURL,
	}))

	// Example ping request.
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
```

## Screenshot

Run app server:

```sh
go run example/main.go
```

Test request:

```sh
curl http://localhost:8080/ping
```

<img src="./images/screen.png">
