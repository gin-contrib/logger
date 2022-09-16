package logger

import (
	"io"
	"net/http"
	"os"
	"regexp"
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
}

// SetLogger initializes the logging middleware.
func SetLogger(opts ...Option) gin.HandlerFunc {
	cfg := &config{
		defaultLevel:     zerolog.InfoLevel,
		clientErrorLevel: zerolog.WarnLevel,
		serverErrorLevel: zerolog.ErrorLevel,
		output:           gin.DefaultWriter,
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
		isTerm := isatty.IsTerminal(os.Stdout.Fd())
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

			l = l.With().
				Int("status", c.Writer.Status()).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Str("ip", c.ClientIP()).
				Dur("latency", latency).
				Str("user_agent", c.Request.UserAgent()).Logger()

			msg := "Request"
			if len(c.Errors) > 0 {
				msg = c.Errors.String()
			}

			switch {
			case c.Writer.Status() >= http.StatusBadRequest && c.Writer.Status() < http.StatusInternalServerError:
				{
					l.WithLevel(cfg.clientErrorLevel).
						Msg(msg)
				}
			case c.Writer.Status() >= http.StatusInternalServerError:
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
