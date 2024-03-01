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

// Skipper is a function to skip logs based on provided Context
type Skipper func(c *gin.Context) bool

// Config defines the config for logger middleware
type config struct {
	logger Fn
	// UTC a boolean stating whether to use UTC time zone or local.
	utc             bool
	skipPath        []string
	skipPathRegexps []*regexp.Regexp
	// skip is a Skipper that indicates which logs should not be written.
	// Optional.
	skip Skipper
	// Output is a writer where logs are written.
	// Optional. Default value is gin.DefaultWriter.
	output io.Writer
	// the log level used for request with status code < 400
	defaultLevel zerolog.Level
	// the log level used for request with status code between 400 and 499
	clientErrorLevel zerolog.Level
	// the log level used for request with status code >= 500
	serverErrorLevel zerolog.Level
	// the log level to use for a specific path with status code < 400
	pathLevels map[string]zerolog.Level
}

var isTerm bool = isatty.IsTerminal(os.Stdout.Fd())

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
	return func(c *gin.Context) {
		rl := l
		if cfg.logger != nil {
			rl = cfg.logger(c, l)
		}

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		c.Next()
		track := true

		if _, ok := skip[path]; ok || (cfg.skip != nil && cfg.skip(c)) {
			track = false
		}

		if track && len(cfg.skipPathRegexps) > 0 {
			for _, reg := range cfg.skipPathRegexps {
				if !reg.MatchString(path) {
					continue
				}

				track = false
				break
			}
		}

		if track {
			end := time.Now()
			if cfg.utc {
				end = end.UTC()
			}
			latency := end.Sub(start)

			msg := "Request"
			if len(c.Errors) > 0 {
				msg = c.Errors.String()
			}

			var evt *zerolog.Event
			level, hasLevel := cfg.pathLevels[path]

			switch {
			case c.Writer.Status() >= http.StatusBadRequest && c.Writer.Status() < http.StatusInternalServerError:
				evt = rl.WithLevel(cfg.clientErrorLevel)
			case c.Writer.Status() >= http.StatusInternalServerError:
				evt = rl.WithLevel(cfg.serverErrorLevel)
			case hasLevel:
				evt = rl.WithLevel(level)
			default:
				evt = rl.WithLevel(cfg.defaultLevel)
			}
			evt.
				Int("status", c.Writer.Status()).
				Str("method", c.Request.Method).
				Str("path", path).
				Str("ip", c.ClientIP()).
				Dur("latency", latency).
				Str("user_agent", c.Request.UserAgent()).
				Int("body_size", c.Writer.Size()).
				Msg(msg)
		}
	}
}

// ParseLevel converts a level string into a zerolog Level value.
// returns an error if the input string does not match known values.
func ParseLevel(levelStr string) (zerolog.Level, error) {
	return zerolog.ParseLevel(levelStr)
}
