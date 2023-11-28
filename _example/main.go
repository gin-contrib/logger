package main

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-contrib/logger"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

var rxURL = regexp.MustCompile(`^/regexp\d*`)

func main() {
	r := gin.New()

	// Add a logger middleware, which:
	//   - Logs all requests, like a combined access and error log.
	//   - Logs to stdout.
	// r.Use(logger.SetLogger())

	// Example pong request.
	r.GET("/pong", logger.SetLogger(), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Example ping request.
	r.GET("/ping", logger.SetLogger(
		logger.WithSkipPath([]string{"/skip"}),
		logger.WithUTC(true),
		logger.WithSkipPathRegexps(rxURL),
	), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Example skip path request.
	r.GET("/skip", logger.SetLogger(
		logger.WithSkipPath([]string{"/skip"}),
	), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Example skip path request.
	r.GET("/regexp1", logger.SetLogger(
		logger.WithSkipPathRegexp(rxURL),
	), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Example skip path request.
	r.GET("/regexp2", logger.SetLogger(
		logger.WithSkipPathRegexp(rxURL),
	), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// add custom fields.
	r.GET("/id", requestid.New(requestid.WithGenerator(func() string {
		return "foobar"
	})), logger.SetLogger(
		logger.WithLogger(func(c *gin.Context, l zerolog.Logger) zerolog.Logger {
			if trace.SpanFromContext(c.Request.Context()).SpanContext().IsValid() {
				l = l.With().
					Str("trace_id", trace.SpanFromContext(c.Request.Context()).SpanContext().TraceID().String()).
					Str("span_id", trace.SpanFromContext(c.Request.Context()).SpanContext().SpanID().String()).
					Logger()
			}

			return l.With().
				Str("id", requestid.Get(c)).
				Str("foo", "bar").
				Str("path", c.Request.URL.Path).
				Logger()
		}),
	), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Example of JSON format log
	r.GET("/json", logger.SetLogger(
		logger.WithLogger(func(_ *gin.Context, l zerolog.Logger) zerolog.Logger {
			return l.Output(gin.DefaultWriter).With().Logger()
		}),
	), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Listen and Server in 0.0.0.0:8080
	if err := r.Run(":8080"); err != nil {
		log.Fatal().Msg("can' start server with 8080 port")
	}
}
