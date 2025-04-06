package logger

import (
	"io"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

/*
Option is an interface that defines a method to apply a configuration
to a given config instance. Implementations of this interface can be
used to modify the configuration settings of the logger.
*/
type Option interface {
	apply(*config)
}

/*
Ensures that optionFunc implements the Option interface at compile time.
If optionFunc does not implement Option, a compile-time error will occur.
*/
var _ Option = (*optionFunc)(nil)

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

/*
WithLogger returns an Option that sets the logger function in the config.
The logger function is a function that takes a *gin.Context and a zerolog.Logger,
and returns a zerolog.Logger. This function is typically used to modify or enhance
the logger within the context of a Gin HTTP request.

Parameters:

	fn - A function that takes a *gin.Context and a zerolog.Logger, and returns a zerolog.Logger.

Returns:

	Option - An option that sets the logger function in the config.
*/
func WithLogger(fn func(*gin.Context, zerolog.Logger) zerolog.Logger) Option {
	return optionFunc(func(c *config) {
		c.logger = fn
	})
}

/*
WithSkipPathRegexps returns an Option that sets the skipPathRegexps field in the config.
The skipPathRegexps field is a list of regular expressions that match paths to be skipped from logging.

Parameters:

	regs - A list of regular expressions to match paths to be skipped from logging.

Returns:

	Option - An option that sets the skipPathRegexps field in the config.
*/
func WithSkipPathRegexps(regs ...*regexp.Regexp) Option {
	return optionFunc(func(c *config) {
		if len(regs) == 0 {
			return
		}

		c.skipPathRegexps = append(c.skipPathRegexps, regs...)
	})
}

/*
WithUTC returns an Option that sets the utc field in the config.
The utc field is a boolean that indicates whether to use UTC time zone or local time zone.

Parameters:

	s - A boolean indicating whether to use UTC time zone.

Returns:

	Option - An option that sets the utc field in the config.
*/
func WithUTC(s bool) Option {
	return optionFunc(func(c *config) {
		c.utc = s
	})
}

/*
WithSkipPath returns an Option that sets the skipPath field in the config.
The skipPath field is a list of URL paths to be skipped from logging.

Parameters:

	s - A list of URL paths to be skipped from logging.

Returns:

	Option - An option that sets the skipPath field in the config.
*/
func WithSkipPath(s []string) Option {
	return optionFunc(func(c *config) {
		c.skipPath = s
	})
}

/*
WithPathLevel returns an Option that sets the pathLevels field in the config.
The pathLevels field is a map that associates specific URL paths with logging levels.

Parameters:

	m - A map where the keys are URL paths and the values are zerolog.Level.

Returns:

	Option - An option that sets the pathLevels field in the config.
*/
func WithPathLevel(m map[string]zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.pathLevels = m
	})
}

/*
WithWriter returns an Option that sets the output field in the config.
The output field is an io.Writer that specifies the destination for log output.

Parameters:

	s - The writer to be used for log output.

Returns:

	Option - An option that sets the output field in the config.
*/
func WithWriter(s io.Writer) Option {
	return optionFunc(func(c *config) {
		c.output = s
	})
}

/*
WithDefaultLevel returns an Option that sets the defaultLevel field in the config.
The defaultLevel field specifies the logging level for requests with status codes less than 400.

Parameters:

	lvl - The logging level to be used for requests with status codes less than 400.

Returns:

	Option - An option that sets the defaultLevel field in the config.
*/
func WithDefaultLevel(lvl zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.defaultLevel = lvl
	})
}

/*
WithClientErrorLevel returns an Option that sets the clientErrorLevel field in the config.
The clientErrorLevel field specifies the logging level for requests with status codes between 400 and 499.

Parameters:

	lvl - The logging level to be used for requests with status codes between 400 and 499.

Returns:

	Option - An option that sets the clientErrorLevel field in the config.
*/
func WithClientErrorLevel(lvl zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.clientErrorLevel = lvl
	})
}

/*
WithServerErrorLevel returns an Option that sets the serverErrorLevel field in the config.
The serverErrorLevel field specifies the logging level for server errors.

Parameters:

	lvl - The logging level to be used for server errors.

Returns:

	Option - An option that sets the serverErrorLevel field in the config.
*/
func WithServerErrorLevel(lvl zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.serverErrorLevel = lvl
	})
}

/*
WithSkipper returns an Option that sets the Skipper function in the config.
The Skipper function determines whether a request should be skipped for logging.

Parameters:

	s - A function that takes a gin.Context and returns a boolean indicating whether the request should be skipped.

Returns:

	Option - An option that sets the Skipper function in the config.
*/
func WithSkipper(s Skipper) Option {
	return optionFunc(func(c *config) {
		c.skip = s
	})
}

/*
WithContext returns an Option that sets the context field in the config.
The context field is a function that takes a *gin.Context and a *zerolog.Event, and returns a modified *zerolog.Event.
This allows for custom logging behavior based on the request context.

Parameters:

	fn - A function that takes a *gin.Context and a *zerolog.Event, and returns a modified *zerolog.Event.

Returns:

	Option - An option that sets the context field in the config.
*/
func WithContext(fn func(*gin.Context, *zerolog.Event) *zerolog.Event) Option {
	return optionFunc(func(c *config) {
		c.context = fn
	})
}

/*
WithMessage returns an Option that sets the message field in the config.
The message field specifies a custom log message to be used when an HTTP request has finished and is logged.

Parameters:

	message - The custom log message.

Returns:

	Option - An option that sets the message field in the config.
*/
func WithMessage(message string) Option {
	return optionFunc(func(c *config) {
		c.message = message
	})
}

/*
WithSpecificLogLevelByStatusCode returns an Option that sets the specificLevelByStatusCode field in the config.
The specificLevelByStatusCode field is a map that associates specific HTTP status codes with logging levels.

Parameters:

	statusCodes - A map where the keys are HTTP status codes and the values are zerolog.Level.

Returns:

	Option - An option that sets the specificLevelByStatusCode field in the config.
*/
func WithSpecificLogLevelByStatusCode(statusCodes map[int]zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.specificLevelByStatusCode = statusCodes
	})
}
