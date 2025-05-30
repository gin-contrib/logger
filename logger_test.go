package logger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
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

	resp := performRequest(r, "GET", "/example?a=100", header{"X-Request-Id", "123"})
	assert.Equal(t, 200, resp.Code)
	assert.Contains(t, buffer.String(), "200")
	assert.Contains(t, buffer.String(), "GET")
	assert.Contains(t, buffer.String(), "/example")
	assert.Contains(t, buffer.String(), "path=/example?a=100")

	buffer.Reset()
	performRequest(r, "POST", "/example?a=100")
	assert.Contains(t, buffer.String(), "400")
	assert.Contains(t, buffer.String(), "POST")
	assert.Contains(t, buffer.String(), "/example")
	assert.Contains(t, buffer.String(), "WRN")
	assert.Contains(t, buffer.String(), "path=/example?a=100")

	buffer.Reset()
	performRequest(r, "PUT", "/example?a=100")
	assert.Contains(t, buffer.String(), "502")
	assert.Contains(t, buffer.String(), "PUT")
	assert.Contains(t, buffer.String(), "/example")
	assert.Contains(t, buffer.String(), "ERR")
	assert.Contains(t, buffer.String(), "path=/example?a=100")

	buffer.Reset()
	r.GET("/example-with-additional-log", func(ctx *gin.Context) {
		l := Get(ctx)
		l.Info().Msg("additional log")
	})
	performRequest(r, "GET", "/example-with-additional-log")
	assert.Contains(t, buffer.String(), "200")
	assert.Contains(t, buffer.String(), "GET")
	assert.Contains(t, buffer.String(), "/example-with-additional-log")
	assert.Contains(t, buffer.String(), "additional log")
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
		WithSkipPathRegexps(rxURL),
	), func(c *gin.Context) {})

	r.GET("/regexp02", SetLogger(
		WithWriter(buffer),
		WithSkipPathRegexps(rxURL),
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

type concurrentBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *concurrentBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func TestCustomLoggerIssue68(t *testing.T) {
	buffer := new(concurrentBuffer)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// Use JSON logger as it will explicitly print keys multiple times if they are added multiple times,
	// which may happen if there are mutations to the logger.
	r.Use(SetLogger(
		WithLogger(func(_ *gin.Context, l zerolog.Logger) zerolog.Logger { return l.Output(buffer).With().Logger() }),
		WithDefaultLevel(zerolog.DebugLevel),
		WithClientErrorLevel(zerolog.ErrorLevel),
		WithServerErrorLevel(zerolog.FatalLevel),
	))
	r.GET("/example", func(c *gin.Context) {})

	// concurrent requests should only have their info logged once
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		req := fmt.Sprintf("/example?a=%d", i)
		go func() {
			defer wg.Done()
			performRequest(r, "GET", req)
		}()
	}
	wg.Wait()

	bs := buffer.b.String()
	for i := 0; i < 10; i++ {
		// should contain each request log exactly once
		msg := fmt.Sprintf("/example?a=%d", i)
		if assert.Contains(t, bs, msg) {
			assert.Equal(t, strings.Index(bs, msg), strings.LastIndex(bs, msg))
		}
	}
}

func TestLoggerParseLevel(t *testing.T) {
	type args struct {
		levelStr string
	}
	tests := []struct {
		name    string
		args    args
		want    zerolog.Level
		wantErr bool
	}{
		{"trace", args{"trace"}, zerolog.TraceLevel, false},
		{"debug", args{"debug"}, zerolog.DebugLevel, false},
		{"info", args{"info"}, zerolog.InfoLevel, false},
		{"warn", args{"warn"}, zerolog.WarnLevel, false},
		{"error", args{"error"}, zerolog.ErrorLevel, false},
		{"fatal", args{"fatal"}, zerolog.FatalLevel, false},
		{"panic", args{"panic"}, zerolog.PanicLevel, false},
		{"disabled", args{"disabled"}, zerolog.Disabled, false},
		{"nolevel", args{""}, zerolog.NoLevel, false},
		{"-1", args{"-1"}, zerolog.TraceLevel, false},
		{"-2", args{"-2"}, zerolog.Level(-2), false},
		{"-3", args{"-3"}, zerolog.Level(-3), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLevel(tt.args.levelStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLevel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseLevel() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoggerCustomLevel(t *testing.T) {
	buffer := new(bytes.Buffer)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(SetLogger(
		WithWriter(buffer),
		WithDefaultLevel(zerolog.InfoLevel),
		WithClientErrorLevel(zerolog.ErrorLevel),
		WithServerErrorLevel(zerolog.FatalLevel),
		WithPathLevel(map[string]zerolog.Level{
			"/example": zerolog.DebugLevel,
		}),
	))
	r.GET("/example", func(c *gin.Context) {})
	r.POST("/example", func(c *gin.Context) {
		c.String(http.StatusBadRequest, "ok")
	})
	r.PUT("/example", func(c *gin.Context) {
		c.String(http.StatusBadGateway, "ok")
	})
	r.GET("/example2", func(c *gin.Context) {})

	performRequest(r, "GET", "/example")
	assert.Contains(t, buffer.String(), "DBG")

	buffer.Reset()
	performRequest(r, "GET", "/example2")
	assert.Contains(t, buffer.String(), "INF")

	buffer.Reset()
	performRequest(r, "POST", "/example")
	assert.Contains(t, buffer.String(), "ERR")

	buffer.Reset()
	performRequest(r, "PUT", "/example")
	assert.Contains(t, buffer.String(), "FTL")
}

func TestLoggerSkipper(t *testing.T) {
	buffer := new(bytes.Buffer)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(SetLogger(
		WithWriter(buffer),
		WithSkipper(func(c *gin.Context) bool {
			return c.Request.URL.Path == "/example2"
		}),
	))
	r.GET("/example", func(c *gin.Context) {})
	r.GET("/example2", func(c *gin.Context) {})

	performRequest(r, "GET", "/example")
	assert.Contains(t, buffer.String(), "GET")
	assert.Contains(t, buffer.String(), "/example")

	buffer.Reset()
	performRequest(r, "GET", "/example2")
	assert.NotContains(t, buffer.String(), "GET")
	assert.NotContains(t, buffer.String(), "/example2")
}

func TestLoggerCustomMessage(t *testing.T) {
	buffer := new(bytes.Buffer)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(SetLogger(
		WithWriter(buffer),
		WithMessage("Custom message"),
	))
	r.GET("/example", func(c *gin.Context) {})

	performRequest(r, "GET", "/example")
	assert.Contains(t, buffer.String(), "Custom message")
}

func TestLoggerCustomMessageWithErrors(t *testing.T) {
	buffer := new(bytes.Buffer)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(SetLogger(
		WithWriter(buffer),
		WithMessage("Custom message"),
	))
	r.GET("/example", func(c *gin.Context) {
		_ = c.Error(errors.New("custom error"))
	})

	performRequest(r, "GET", "/example")
	assert.Contains(t, buffer.String(), "Custom message with errors: ")
	assert.Equal(t, strings.Count(buffer.String(), " with errors: "), 1)

	// Reset and test again to make sure we're not appending to the existing error message
	buffer.Reset()
	performRequest(r, "GET", "/example")
	assert.Contains(t, buffer.String(), "Custom message with errors: ")
	assert.Equal(t, strings.Count(buffer.String(), " with errors: "), 1)
}

func BenchmarkLogger(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(SetLogger(WithDefaultLevel(zerolog.Disabled)))
	r.GET("/", func(ctx *gin.Context) {
		ctx.Data(200, "text/plain", []byte("all good"))
	})

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		req, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
		if err != nil {
			b.Errorf("NewRequestWithContext() error = %v", err)
			return
		}
		w := httptest.NewRecorder()

		for pb.Next() {
			r.ServeHTTP(w, req)
		}
	})
}
