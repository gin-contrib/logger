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

/*
Fn is a function type that takes a gin.Context and a zerolog.Logger as parameters,
and returns a zerolog.Logger. It is typically used to modify or enhance the logger
within the context of a Gin HTTP request.
*/
type Fn func(*gin.Context, zerolog.Logger) zerolog.Logger

/*
EventFn is a function type that takes a gin.Context and a zerolog.Event as parameters,
and returns a zerolog.Event. It is typically used to modify or enhance the event
within the context of a Gin HTTP request.
*/
type EventFn func(*gin.Context, *zerolog.Event) *zerolog.Event

/*
Skipper defines a function to skip middleware. It takes a gin.Context as input
and returns a boolean indicating whether to skip the middleware for the given context.
*/
type Skipper func(c *gin.Context) bool

/*
config holds the configuration for the logger middleware.
*/
type config struct {
	/*
		logger is a function that defines the logging behavior.
	*/
	logger Fn
	/*
		context is a function that defines the logging behavior of gin.Context data
	*/
	context EventFn
	/*
		utc is a boolean stating whether to use UTC time zone or local.
	*/
	utc bool
	/*
		skipPath is a list of paths to be skipped from logging.
	*/
	skipPath []string
	/*
		skipPathRegexps is a list of regular expressions to match paths to be skipped from logging.
	*/
	skipPathRegexps []*regexp.Regexp
	/*
		skip is a Skipper that indicates which logs should not be written. Optional.
	*/
	skip Skipper
	/*
		output is a writer where logs are written. Optional. Default value is gin.DefaultWriter.
	*/
	output io.Writer
	/*
		defaultLevel is the log level used for requests with status code < 400.
	*/
	defaultLevel zerolog.Level
	/*
		clientErrorLevel is the log level used for requests with status code between 400 and 499.
	*/
	clientErrorLevel zerolog.Level
	/*
		serverErrorLevel is the log level used for requests with status code >= 500.
	*/
	serverErrorLevel zerolog.Level
	/*
		pathLevels is a map of specific paths to log levels for requests with status code < 400.
	*/
	pathLevels map[string]zerolog.Level
	/*
		message is a custom string that sets a log-message when http-request has finished
	*/
	message string
	/*
		specificLevelByStatusCode is a map of specific status codes to log levels every request
	*/
	specificLevelByStatusCode map[int]zerolog.Level
}

const loggerKey = "_gin-contrib/logger_"

var isTerm = isatty.IsTerminal(os.Stdout.Fd())

/*
SetLogger returns a gin.HandlerFunc (middleware) that logs requests using zerolog.
It accepts a variadic number of Option functions to customize the logger's behavior.

The logger configuration includes:
  - defaultLevel: the default logging level (default: zerolog.InfoLevel).
  - clientErrorLevel: the logging level for client errors (default: zerolog.WarnLevel).
  - serverErrorLevel: the logging level for server errors (default: zerolog.ErrorLevel).
  - output: the output writer for the logger (default: gin.DefaultWriter).
  - skipPath: a list of paths to skip logging.
  - skipPathRegexps: a list of regular expressions to skip logging for matching paths.
  - logger: a custom logger function to use instead of the default logger.

The middleware logs the following request details:
  - method: the HTTP method of the request.
  - path: the URL path of the request.
  - ip: the client's IP address.
  - user_agent: the User-Agent header of the request.
  - status: the HTTP status code of the response.
  - latency: the time taken to process the request.
  - body_size: the size of the response body.

The logging level for each request is determined based on the response status code:
  - clientErrorLevel for 4xx status codes.
  - serverErrorLevel for 5xx status codes.
  - defaultLevel for other status codes.
  - Custom levels can be set for specific paths using the pathLevels configuration.
*/
func SetLogger(opts ...Option) gin.HandlerFunc {
	cfg := &config{
		defaultLevel:     zerolog.InfoLevel,
		clientErrorLevel: zerolog.WarnLevel,
		serverErrorLevel: zerolog.ErrorLevel,
		output:           os.Stderr,
		message:          "Request",
	}

	// Apply each option to the config
	for _, o := range opts {
		o.apply(cfg)
	}

	// Create a set of paths to skip logging
	skip := make(map[string]struct{}, len(cfg.skipPath))
	for _, path := range cfg.skipPath {
		skip[path] = struct{}{}
	}

	// Initialize the base logger
	l := zerolog.New(cfg.output).
		Output(zerolog.ConsoleWriter{Out: cfg.output, NoColor: !isTerm}).
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
		if raw := c.Request.URL.RawQuery; raw != "" {
			path += "?" + raw
		}

		track := !shouldSkipLogging(path, skip, cfg, c)

		contextLogger := rl
		if track {
			contextLogger = rl.With().
				Str("method", c.Request.Method).
				Str("path", path).
				Str("ip", c.ClientIP()).
				Str("user_agent", c.Request.UserAgent()).
				Logger()
		}
		c.Set(loggerKey, contextLogger)

		c.Next()

		if track {
			end := time.Now()
			if cfg.utc {
				end = end.UTC()
			}
			latency := end.Sub(start)

			msg := cfg.message
			if len(c.Errors) > 0 {
				msg += " with errors: " + c.Errors.String()
			}

			evt := getLogEvent(rl, cfg, c, path)

			if cfg.context != nil {
				evt = cfg.context(c, evt)
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

/*
ParseLevel parses a string representation of a log level and returns the corresponding zerolog.Level.
It takes a single argument:
  - levelStr: a string representing the log level (e.g., "debug", "info", "warn", "error").

It returns:
  - zerolog.Level: the parsed log level.
  - error: an error if the log level string is invalid.
*/
func ParseLevel(levelStr string) (zerolog.Level, error) {
	return zerolog.ParseLevel(levelStr)
}

func shouldSkipLogging(path string, skip map[string]struct{}, cfg *config, c *gin.Context) bool {
	if _, ok := skip[path]; ok || (cfg.skip != nil && cfg.skip(c)) {
		return true
	}
	for _, reg := range cfg.skipPathRegexps {
		if reg.MatchString(path) {
			return true
		}
	}
	return false
}

func getLogEvent(rl zerolog.Logger, cfg *config, c *gin.Context, path string) *zerolog.Event {
	level, hasLevel := cfg.pathLevels[path]
	specificLogLevel, hasSpecificLogLevel := cfg.specificLevelByStatusCode[c.Writer.Status()]

	switch {
	case hasSpecificLogLevel:
		return rl.WithLevel(specificLogLevel).Ctx(c)
	case c.Writer.Status() >= http.StatusBadRequest && c.Writer.Status() < http.StatusInternalServerError:
		return rl.WithLevel(cfg.clientErrorLevel).Ctx(c)
	case c.Writer.Status() >= http.StatusInternalServerError:
		return rl.WithLevel(cfg.serverErrorLevel).Ctx(c)
	case hasLevel:
		return rl.WithLevel(level).Ctx(c)
	default:
		return rl.WithLevel(cfg.defaultLevel).Ctx(c)
	}
}

/*
GetLogger retrieves the zerolog.Logger instance from the given gin.Context.
It assumes that the logger has been previously set in the context with the key loggerKey.
If the logger is not found, it will panic.

Parameters:

	c - the gin.Context from which to retrieve the logger.

Returns:

	zerolog.Logger - the logger instance stored in the context.
*/
func Get(c *gin.Context) zerolog.Logger {
	return c.MustGet(loggerKey).(zerolog.Logger)
}
