package logger

import (
	"io"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Option is an interface that defines a method to apply a configuration
// to a given config instance. Implementations of this interface can be
// used to modify the configuration settings of the logger.
type Option interface {
	apply(*config)
}

// Ensures that optionFunc implements the Option interface at compile time.
// If optionFunc does not implement Option, a compile-time error will occur.
var _ Option = (*optionFunc)(nil)

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// WithLogger returns an Option that sets the logger function in the config.
// The logger function is a function that takes a *gin.Context and a zerolog.Logger,
// and returns a zerolog.Logger. This function is typically used to modify or enhance
// the logger within the context of a Gin HTTP request.
//
// Parameters:
//
//	fn (Fn): A function that takes a *gin.Context and a zerolog.Logger, and returns a zerolog.Logger.
//
// Returns:
//
//	Option: An option that sets the logger function in the config.
func WithLogger(fn func(*gin.Context, zerolog.Logger) zerolog.Logger) Option {
	return optionFunc(func(c *config) {
		c.logger = fn
	})
}

// WithSkipPathRegexps returns an Option that sets the skipPathRegexps field in the config.
// The skipPathRegexps field is a list of regular expressions that match paths to be skipped from logging.
//
// Parameters:
//
//	regs ([]*regexp.Regexp): A list of regular expressions to match paths to be skipped from logging.
//
// Returns:
//
//	Option: An option that sets the skipPathRegexps field in the config.
func WithSkipPathRegexps(regs ...*regexp.Regexp) Option {
	return optionFunc(func(c *config) {
		if len(regs) == 0 {
			return
		}

		c.skipPathRegexps = append(c.skipPathRegexps, regs...)
	})
}

// WithUTC returns an Option that sets the utc field in the config.
// The utc field is a boolean that indicates whether to use UTC time zone or local time zone.
//
// Parameters:
//
//	s (bool): A boolean indicating whether to use UTC time zone.
//
// Returns:
//
//	Option: An option that sets the utc field in the config.
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
