package logger

import (
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Config defines the config for logger middleware
type Config struct {
	Logger *zerolog.Logger
	// UTC a boolean stating whether to use UTC time zone or local.
	UTC            bool
	SkipPath       []string
	SkipPathRegexp *regexp.Regexp
}

// SetLogger initializes the logging middleware.
func SetLogger(config ...Config) gin.HandlerFunc {
	var newConfig Config
	if len(config) > 0 {
		newConfig = config[0]
	}
	var skip map[string]struct{}
	if length := len(newConfig.SkipPath); length > 0 {
		skip = make(map[string]struct{}, length)
		for _, path := range newConfig.SkipPath {
			skip[path] = struct{}{}
		}
	}

	var sublog zerolog.Logger
	if newConfig.Logger == nil {
		sublog = log.Logger
	} else {
		sublog = *newConfig.Logger
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
			newConfig.SkipPathRegexp != nil &&
			newConfig.SkipPathRegexp.MatchString(path) {
			track = false
		}

		if track {
			end := time.Now()
			latency := end.Sub(start)
			if newConfig.UTC {
				end = end.UTC()
			}

			msg := "Request"
			if len(c.Errors) > 0 {
				msg = c.Errors.String()
			}

			dumplogger := sublog.With().
				Int("status", c.Writer.Status()).
				Str("method", c.Request.Method).
				Str("path", path).
				Str("ip", c.ClientIP()).
				Dur("latency", latency).
				Str("user_agent", c.Request.UserAgent()).
				Logger()

			switch {
			case c.Writer.Status() >= http.StatusBadRequest && c.Writer.Status() < http.StatusInternalServerError:
				{
					dumplogger.Warn().
						Msg(msg)
				}
			case c.Writer.Status() >= http.StatusInternalServerError:
				{
					dumplogger.Error().
						Msg(msg)
				}
			default:
				dumplogger.Info().
					Msg(msg)
			}
		}
	}
}
