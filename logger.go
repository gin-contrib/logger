package logger

import (
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func defaultLogger(c *gin.Context, out io.Writer, latency time.Duration) zerolog.Logger {
	logger := zerolog.New(out).With().
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

// Option for timeout
type Option func(*config)

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
}

// WithLogger set custom logger func
func WithLogger(fn func(*gin.Context, io.Writer, time.Duration) zerolog.Logger) Option {
	return func(c *config) {
		c.logger = fn
	}
}

// WithSkipPathRegexp skip URL path by regexp pattern
func WithSkipPathRegexp(s *regexp.Regexp) Option {
	return func(c *config) {
		c.skipPathRegexp = s
	}
}

// WithUTC returns t with the location set to UTC.
func WithUTC(s bool) Option {
	return func(c *config) {
		c.utc = s
	}
}

// WithSkipPath skip URL path by specfic pattern
func WithSkipPath(s []string) Option {
	return func(c *config) {
		c.skipPath = s
	}
}

// WithSkipPath skip URL path by specfic pattern
func WithWriter(s io.Writer) Option {
	return func(c *config) {
		c.output = s
	}
}

// SetLogger initializes the logging middleware.
func SetLogger(opts ...Option) gin.HandlerFunc {
	l := &config{
		logger: defaultLogger,
		output: gin.DefaultWriter,
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt(l)
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
					logger.Warn().
						Msg(msg)
				}
			case c.Writer.Status() >= http.StatusInternalServerError:
				{
					logger.Error().
						Msg(msg)
				}
			default:
				logger.Info().
					Msg(msg)
			}
		}
	}
}
