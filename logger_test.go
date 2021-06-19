package logger

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
	r := gin.New()
	r.Use(SetLogger(WithWriter(buffer)))
	r.GET("/example", func(c *gin.Context) {})
	r.POST("/example", func(c *gin.Context) {})
	r.PUT("/example", func(c *gin.Context) {})
	r.DELETE("/example", func(c *gin.Context) {})
	r.PATCH("/example", func(c *gin.Context) {})
	r.HEAD("/example", func(c *gin.Context) {})
	r.OPTIONS("/example", func(c *gin.Context) {})

	performRequest(r, "GET", "/example?a=100")
	assert.Contains(t, buffer.String(), "200")
	assert.Contains(t, buffer.String(), "GET")
	assert.Contains(t, buffer.String(), "/example")
}
