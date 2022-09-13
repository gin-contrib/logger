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

func defaultLogger(c *gin.Context, out io.Writer, latency time.Duration) zerolog.Logger {
	isTerm := isatty.IsTerminal(os.Stdout.Fd())
	logger := zerolog.New(out).
		Output(
			zerolog.ConsoleWriter{
				Out:     out,
				NoColor: !isTerm,
			},
		).
		With().
		Timestamp().
		Int("status", c.Writer.Status()).
		Str("method", c.Request.Method).
		Str("path", c.Request.URL.Path).
		Str("ip", c.ClientIP()).
		Dur("latency", latency).
		Str("user_agent", c.Request.UserAgent()).
		Logger()

	return logger
}

// Config defines the config for logger middleware
type config struct {
	logger func(*gin.Context, io.Writer, time.Duration) zerolog.Logger
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
	l := &config{
		logger:           defaultLogger,
		defaultLevel:     zerolog.InfoLevel,
		clientErrorLevel: zerolog.WarnLevel,
		serverErrorLevel: zerolog.ErrorLevel,
		output:           gin.DefaultWriter,
	}

	// Loop through each option
	for _, o := range opts {
		// Call the option giving the instantiated
		o.apply(l)
	}

	var skip map[string]struct{}
	if length := len(l.skipPath); length > 0 {
		skip = make(map[string]struct{}, length)
		for _, path := range l.skipPath {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
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
			l.skipPathRegexp != nil &&
			l.skipPathRegexp.MatchString(path) {
			track = false
		}

		if track {
			end := time.Now()
			if l.utc {
				end = end.UTC()
			}
			latency := end.Sub(start)
			logger := l.logger(c, l.output, latency)

			msg := "Request"
			if len(c.Errors) > 0 {
				msg = c.Errors.String()
			}

			switch {
			case c.Writer.Status() >= http.StatusBadRequest && c.Writer.Status() < http.StatusInternalServerError:
				{
					logger.WithLevel(l.clientErrorLevel).
						Msg(msg)
				}
			case c.Writer.Status() >= http.StatusInternalServerError:
				{
					logger.WithLevel(l.serverErrorLevel).
						Msg(msg)
				}
			default:
				logger.WithLevel(l.defaultLevel).
					Msg(msg)
			}
		}
	}
}
