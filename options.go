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

// WithSkipPathRegexp skip URL path by regexp pattern
func WithSkipPathRegexp(reg *regexp.Regexp) Option {
	return optionFunc(func(c *config) {
		if reg == nil {
			return
		}

		c.skipPathRegexps = append(c.skipPathRegexps, reg)
	})
}

// WithSkipPathRegexps multiple skip URL paths by regexp pattern
func WithSkipPathRegexps(regs []*regexp.Regexp) Option {
	return optionFunc(func(c *config) {
		if regs == nil {
			return
		}

		for _, reg := range regs {
			c.skipPathRegexps = append(c.skipPathRegexps, reg)
		}
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

// WithWriter change the default output writer.
// Default is gin.DefaultWriter
func WithWriter(s io.Writer) Option {
	return optionFunc(func(c *config) {
		c.output = s
	})
}

func WithDefaultLevel(lvl zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.defaultLevel = lvl
	})
}

func WithClientErrorLevel(lvl zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.clientErrorLevel = lvl
	})
}

func WithServerErrorLevel(lvl zerolog.Level) Option {
	return optionFunc(func(c *config) {
		c.serverErrorLevel = lvl
	})
}
