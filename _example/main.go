package main

import (
	"errors"
	"fmt"
	"math/rand"
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

// main initializes the Gin router, sets up various routes with different logging
// configurations, and starts the server on port 8080. The routes demonstrate
// different ways to use the logger middleware, including logging all requests,
// skipping certain paths, adding custom fields, and using JSON format logs.
//
// Routes:
// - GET /pong: Logs request with default logger settings.
// - GET /ping: Logs request with custom settings, including skipping paths and using UTC time.
// - GET /skip: Logs request but skips logging for the /skip path.
// - GET /regexp1: Logs request but skips logging for paths matching the provided regex.
// - GET /regexp2: Logs request but skips logging for paths matching the provided regex.
// - GET /id: Logs request with custom fields including trace ID, span ID, and a custom request ID.
// - GET /json: Logs request in JSON format.
// - GET /health: Skips logging for the /health path.
// - GET /v1/ping: Skips logging for GET requests in the /v1 group.
// - POST /v1/ping: Logs request for POST requests in the /v1 group.
//
// The server listens on 0.0.0.0:8080.
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
		logger.WithSkipPathRegexps(rxURL),
	), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Example skip path request.
	r.GET("/regexp2", logger.SetLogger(
		logger.WithSkipPathRegexps(rxURL),
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

	// Example of a custom message to the log
	r.GET("/message", logger.SetLogger(
		logger.WithLogger(func(_ *gin.Context, l zerolog.Logger) zerolog.Logger {
			return l.Output(gin.DefaultWriter).With().Logger()
		}),
		logger.WithMessage("Request ended"),
	), func(c *gin.Context) {
		c.Error(errors.New("some error has occured here"))
		c.Error(errors.New("and some error has occured there"))
		c.String(http.StatusBadGateway, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Example of skipper usage
	r.GET("/health", logger.SetLogger(
		logger.WithSkipper(func(c *gin.Context) bool {
			return c.Request.URL.Path == "/health"
		}),
	), func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Example of logging data on gin.Context
	r.GET("/context", logger.SetLogger(
		logger.WithContext(func(c *gin.Context, e *zerolog.Event) *zerolog.Event {
			return e.Any("data1", c.MustGet("data1")).Any("data2", c.MustGet("data2"))
		}),
	), func(c *gin.Context) {
		c.Set("data1", rand.Intn(100))
		c.Set("data2", rand.Intn(100))
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Example of skipper usage
	v1 := r.Group("/v1", logger.SetLogger(
		logger.WithSkipper(func(c *gin.Context) bool {
			return c.Request.Method == "GET"
		})))
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.String(http.StatusOK, "pong01 "+fmt.Sprint(time.Now().Unix()))
		})
		v1.POST("/ping", func(c *gin.Context) {
			c.String(http.StatusOK, "pong02 "+fmt.Sprint(time.Now().Unix()))
		})
	}

	// Listen and Server in 0.0.0.0:8080
	if err := r.Run(":8080"); err != nil {
		log.Fatal().Msg("can' start server with 8080 port")
	}
}
