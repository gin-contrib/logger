package logger

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

type Fn func(*gin.Context, zerolog.Logger) zerolog.Logger

// Config defines the config for logger middleware
type config struct {
	logger Fn
	// UTC a boolean stating whether to use UTC time zone or local.
	utc            bool
	skipPath       []string
	skipPathRegexp *regexp.Regexp
	// Output is a writer where logs are written.
	// Optional. Default value is gin.DefaultWriter.
	output io.Writer
	// the log level used for request with status code < 400
	defaultLevel zerolog.Level
	// the log level used for request with status code between 400 and 499
	clientErrorLevel zerolog.Level
	// the log level used for request with status code >= 500
	serverErrorLevel zerolog.Level
	// whether to log response body for request with status code >= 400
	logErrorResponseBody bool
	// whether to log response body for request with status code < 400
	logResponseBody bool
	// max len of response body message (whatever the status code)
	maxResponseBodyLen int
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

var isTerm bool = isatty.IsTerminal(os.Stdout.Fd())

// SetLogger initializes the logging middleware.
func SetLogger(opts ...Option) gin.HandlerFunc {
	cfg := &config{
		defaultLevel:         zerolog.InfoLevel,
		clientErrorLevel:     zerolog.WarnLevel,
		serverErrorLevel:     zerolog.ErrorLevel,
		output:               gin.DefaultWriter,
		logErrorResponseBody: false,
		logResponseBody:      false,
		maxResponseBodyLen:   50,
	}

	// Loop through each option
	for _, o := range opts {
		// Call the option giving the instantiated
		o.apply(cfg)
	}

	var skip map[string]struct{}
	if length := len(cfg.skipPath); length > 0 {
		skip = make(map[string]struct{}, length)
		for _, path := range cfg.skipPath {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		l := zerolog.New(cfg.output).
			Output(
				zerolog.ConsoleWriter{
					Out:     cfg.output,
					NoColor: !isTerm,
				},
			).
			With().
			Timestamp().
			Logger()

		if cfg.logger != nil {
			l = cfg.logger(c, l)
		}

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		var blw *bodyLogWriter
		if cfg.logErrorResponseBody || cfg.logResponseBody {
			blw = &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
			c.Writer = blw
		}
		c.Next()
		track := true

		if _, ok := skip[path]; ok {
			track = false
		}

		if track &&
			cfg.skipPathRegexp != nil &&
			cfg.skipPathRegexp.MatchString(path) {
			track = false
		}

		if track {
			end := time.Now()
			if cfg.utc {
				end = end.UTC()
			}
			latency := end.Sub(start)

			statusCode := c.Writer.Status()
			var response string
			withResponse := (cfg.logErrorResponseBody && statusCode >= 400) || (cfg.logResponseBody && statusCode < 400)
			if withResponse && blw.body != nil {
				response = blw.body.String()
				response = strings.TrimPrefix(response, "\"")
				response = strings.TrimSuffix(response, "\"")
				if len(response) > cfg.maxResponseBodyLen {
					response = response[:cfg.maxResponseBodyLen] + "..."
				}
			}

			ctx := l.With().
				Int("status", statusCode).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path)
			if withResponse {
				ctx = ctx.Logger().With().Str("response", response)
			}
			l = ctx.Logger().With().
				Str("ip", c.ClientIP()).
				Dur("latency", latency).
				Str("user_agent", c.Request.UserAgent()).Logger()

			msg := "Request"
			if len(c.Errors) > 0 {
				msg = c.Errors.String()
			}

			switch {
			case statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError:
				{
					l.WithLevel(cfg.clientErrorLevel).
						Msg(msg)
				}
			case statusCode >= http.StatusInternalServerError:
				{
					l.WithLevel(cfg.serverErrorLevel).
						Msg(msg)
				}
			default:
				l.WithLevel(cfg.defaultLevel).
					Msg(msg)
			}
		}
	}
}

// ParseLevel converts a level string into a zerolog Level value.
// returns an error if the input string does not match known values.
func ParseLevel(levelStr string) (zerolog.Level, error) {
	return zerolog.ParseLevel(levelStr)
}
