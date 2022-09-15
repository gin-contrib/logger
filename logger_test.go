package logger

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type header struct {
	Key   string
	Value string
}

func performRequest(r http.Handler, method, path string, headers ...header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for _, h := range headers {
		req.Header.Add(h.Key, h.Value)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLogger(t *testing.T) {
	buffer := new(bytes.Buffer)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(SetLogger(WithWriter(buffer)))
	r.GET("/example", func(c *gin.Context) {})
	r.POST("/example", func(c *gin.Context) {
		c.String(http.StatusBadRequest, "ok")
	})
	r.PUT("/example", func(c *gin.Context) {
		c.String(http.StatusBadGateway, "ok")
	})
	r.DELETE("/example", func(c *gin.Context) {})
	r.PATCH("/example", func(c *gin.Context) {})
	r.HEAD("/example", func(c *gin.Context) {})
	r.OPTIONS("/example", func(c *gin.Context) {})

	performRequest(r, "GET", "/example?a=100")
	assert.Contains(t, buffer.String(), "200")
	assert.Contains(t, buffer.String(), "GET")
	assert.Contains(t, buffer.String(), "/example")

	buffer.Reset()
	performRequest(r, "POST", "/example?a=100")
	assert.Contains(t, buffer.String(), "400")
	assert.Contains(t, buffer.String(), "POST")
	assert.Contains(t, buffer.String(), "/example")
	assert.Contains(t, buffer.String(), "WRN")

	buffer.Reset()
	performRequest(r, "PUT", "/example?a=100")
	assert.Contains(t, buffer.String(), "502")
	assert.Contains(t, buffer.String(), "PUT")
	assert.Contains(t, buffer.String(), "/example")
	assert.Contains(t, buffer.String(), "ERR")
}

func TestLoggerWithLogger(t *testing.T) {
	buffer := new(bytes.Buffer)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/example", SetLogger(
		WithWriter(buffer),
		WithUTC(true),
		WithLogger(Fn(func(c *gin.Context, l zerolog.Logger) zerolog.Logger {
			return l.With().
				Str("foo", "bar").
				Str("path", c.Request.URL.Path).
				Logger()
		})),
	), func(c *gin.Context) {})

	r.GET("/example2", SetLogger(
		WithWriter(buffer),
		WithSkipPath([]string{"/example2"}),
	), func(c *gin.Context) {})

	rxURL := regexp.MustCompile(`^/regexp\d*`)

	r.GET("/regexp01", SetLogger(
		WithWriter(buffer),
		WithSkipPathRegexp(rxURL),
	), func(c *gin.Context) {})

	r.GET("/regexp02", SetLogger(
		WithWriter(buffer),
		WithSkipPathRegexp(rxURL),
	), func(c *gin.Context) {})

	performRequest(r, "GET", "/example?a=100")
	assert.Contains(t, buffer.String(), "foo")
	assert.Contains(t, buffer.String(), "bar")
	assert.Contains(t, buffer.String(), "/example")

	buffer.Reset()
	performRequest(r, "GET", "/example2")
	assert.NotContains(t, buffer.String(), "foo")
	assert.NotContains(t, buffer.String(), "bar")
	assert.NotContains(t, buffer.String(), "/example2")

	buffer.Reset()
	performRequest(r, "GET", "/regexp01")
	assert.NotContains(t, buffer.String(), "/regexp01")

	buffer.Reset()
	performRequest(r, "GET", "/regexp02")
	assert.NotContains(t, buffer.String(), "/regexp02")
}

func TestLoggerWithLevels(t *testing.T) {
	buffer := new(bytes.Buffer)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(SetLogger(
		WithWriter(buffer),
		WithDefaultLevel(zerolog.DebugLevel),
		WithClientErrorLevel(zerolog.ErrorLevel),
		WithServerErrorLevel(zerolog.FatalLevel),
	))
	r.GET("/example", func(c *gin.Context) {})
	r.POST("/example", func(c *gin.Context) {
		c.String(http.StatusBadRequest, "ok")
	})
	r.PUT("/example", func(c *gin.Context) {
		c.String(http.StatusBadGateway, "ok")
	})

	performRequest(r, "GET", "/example?a=100")
	assert.Contains(t, buffer.String(), "DBG")

	buffer.Reset()
	performRequest(r, "POST", "/example?a=100")
	assert.Contains(t, buffer.String(), "ERR")

	buffer.Reset()
	performRequest(r, "PUT", "/example?a=100")
	assert.Contains(t, buffer.String(), "FTL")
}
