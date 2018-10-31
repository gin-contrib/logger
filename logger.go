package logger

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// SetLogger initializes the logging middleware.
func SetLogger(logger ...zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		msg := "Request"
		if len(c.Errors) > 0 {
			msg = c.Errors.String()
		}

		var sublog zerolog.Logger
		if len(logger) == 0 {
			sublog = log.Logger
		} else {
			sublog = logger[0]
		}

		dumplogger := sublog.With().
			Int("status", c.Writer.Status()).
			Str("method", c.Request.Method).
			Str("path", path).
			Str("ip", c.ClientIP()).
			Dur("latency", latency).
			Str("user-agent", c.Request.UserAgent()).
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
