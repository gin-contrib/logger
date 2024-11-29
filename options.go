package logger

import (
	"io"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*config)
}

var _ Option = (*optionFunc)(nil)

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithLogger set custom logger func
func WithLogger(fn func(*gin.Context, zerolog.Logger) zerolog.Logger) Option {
	return optionFunc(func(c *config) {
		c.logger = fn
	})
}

// WithSkipPathRegexps multiple skip URL paths by regexp pattern
func WithSkipPathRegexps(regs ...*regexp.Regexp) Option {
	return optionFunc(func(c *config) {
		if len(regs) == 0 {
			return
		}

		c.skipPathRegexps = append(c.skipPathRegexps, regs...)
	})
}

// WithUTC returns t with the location set to UTC.
func WithUTC(s bool) Option {
	return optionFunc(func(c *config) {
		c.utc = s
	})
}

// WithSkipPath skip URL path by specific pattern
func WithSkipPath(s []string) Option {
	return optionFunc(func(c *config) {
		c.skipPath = s
	})
}

// WithPathLevel use logging level for successful requests to a specific path
func WithPathLevel(m map[string]zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.pathLevels = m
	})
}

// WithWriter change the default output writer.
// Default is gin.DefaultWriter
func WithWriter(s io.Writer) Option {
	return optionFunc(func(c *config) {
		c.output = s
	})
}

// WithDefaultLevel set the log level used for request with status code < 400
func WithDefaultLevel(lvl zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.defaultLevel = lvl
	})
}

// WithClientErrorLevel set the log level used for request with status code between 400 and 499
func WithClientErrorLevel(lvl zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.clientErrorLevel = lvl
	})
}

// WithServerErrorLevel sets the logging level for server errors.
// It takes a zerolog.Level as an argument and returns an Option.
// This option modifies the serverErrorLevel field in the config struct.
func WithServerErrorLevel(lvl zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.serverErrorLevel = lvl
	})
}

// WithSkipper returns an Option that sets the Skipper function in the config.
// The Skipper function determines whether a request should be skipped for logging.
//
// Parameters:
//
//	s (Skipper): A function that takes a gin.Context and returns a boolean indicating
//	             whether the request should be skipped.
//
// Returns:
//
//	Option: An option that sets the Skipper function in the config.
func WithSkipper(s Skipper) Option {
	return optionFunc(func(c *config) {
		c.skip = s
	})
}

// WithContext is an option for configuring the logger with a custom context function.
// The provided function takes a *gin.Context and a *zerolog.Event, and returns a modified *zerolog.Event.
// This allows for custom logging behavior based on the request context.
//
// Parameters:
//
//	fn - A function that takes a *gin.Context and a *zerolog.Event, and returns a modified *zerolog.Event.
//
// Returns:
//
//	An Option that applies the custom context function to the logger configuration.
func WithContext(fn func(*gin.Context, *zerolog.Event) *zerolog.Event) Option {
	return optionFunc(func(c *config) {
		c.context = fn
	})
}
