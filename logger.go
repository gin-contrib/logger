package logger

import (
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func defaultLogger(c *gin.Context, latency time.Duration) zerolog.Logger {
	logger := log.Logger.With().
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
type Option func(*Config)

// Config defines the config for logger middleware
type Config struct {
	Logger func(*gin.Context, time.Duration) zerolog.Logger
	// UTC a boolean stating whether to use UTC time zone or local.
	UTC            bool
	SkipPath       []string
	SkipPathRegexp *regexp.Regexp
}

func WithLogger(fn func(*gin.Context, time.Duration) zerolog.Logger) Option {
	return func(c *Config) {
		c.Logger = fn
	}
}

func WithSkipPathRegexp(s *regexp.Regexp) Option {
	return func(c *Config) {
		c.SkipPathRegexp = s
	}
}

func WithUTC(s bool) Option {
	return func(c *Config) {
		c.UTC = s
	}
}

func WithSkipPath(s []string) Option {
	return func(c *Config) {
		c.SkipPath = s
	}
}

// SetLogger initializes the logging middleware.
func SetLogger(opts ...Option) gin.HandlerFunc {
	l := &Config{
		Logger: defaultLogger,
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		opt(l)
	}

	var skip map[string]struct{}
	if length := len(l.SkipPath); length > 0 {
		skip = make(map[string]struct{}, length)
		for _, path := range l.SkipPath {
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
			l.SkipPathRegexp != nil &&
			l.SkipPathRegexp.MatchString(path) {
			track = false
		}

		if track {
			end := time.Now()
			if l.UTC {
				end = end.UTC()
			}
			latency := end.Sub(start)
			logger := l.Logger(c, latency)

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
