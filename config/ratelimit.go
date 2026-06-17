package config

import (
	"errors"
	"time"

	"azugo.io/core/cache"
	"azugo.io/core/ratelimit"
	"azugo.io/core/validation"
	"github.com/spf13/viper"
)

// RateLimit configuration for routes.
type RateLimit struct {
	Enabled bool `mapstructure:"enabled"`

	Strategy string `mapstructure:"strategy" validate:"required,oneof=fixed-window token-bucket"`
	// Fixed window strategy.
	Limit  int           `mapstructure:"limit" validate:"required_if=Strategy fixed-window,omitempty,gt=0"`
	Window time.Duration `mapstructure:"window" validate:"required_if=Strategy fixed-window,omitempty,gt=0"`
	// Token bucket strategy.
	Rate  float64 `mapstructure:"rate" validate:"required_if=Strategy token-bucket,omitempty,gt=0"`
	Burst int     `mapstructure:"burst" validate:"required_if=Strategy token-bucket,omitempty,gt=0"`

	WaitLimit time.Duration `mapstructure:"wait_limit" validate:"omitempty,min=0"`
}

// Bind rate limiter configuration section.
func (c *RateLimit) Bind(prefix string, v *viper.Viper) {
	v.SetDefault(prefix+".enabled", false)
	v.SetDefault(prefix+".strategy", "fixed-window")
	v.SetDefault(prefix+".limit", 60)
	v.SetDefault(prefix+".window", time.Minute)
	v.SetDefault(prefix+".rate", 1)
	v.SetDefault(prefix+".burst", 60)

	_ = v.BindEnv(prefix+".enabled", "RATELIMIT_ENABLED")
	_ = v.BindEnv(prefix+".strategy", "RATELIMIT_STRATEGY")
	_ = v.BindEnv(prefix+".limit", "RATELIMIT_LIMIT")
	_ = v.BindEnv(prefix+".window", "RATELIMIT_WINDOW")
	_ = v.BindEnv(prefix+".rate", "RATELIMIT_RATE")
	_ = v.BindEnv(prefix+".burst", "RATELIMIT_BURST")
	_ = v.BindEnv(prefix+".wait_limit", "RATELIMIT_WAIT_LIMIT")
}

// Validate rate limiter configuration section.
func (c *RateLimit) Validate(valid *validation.Validate) error {
	if !c.Enabled {
		return nil
	}

	return valid.Struct(c)
}

// New creates a limiter from the configuration. Additional options (for
// example ratelimit.Instrumenter) are appended after the configuration-derived
// ones.
func (c *RateLimit) New(cache *cache.Cache, name string, opts ...ratelimit.LimiterOption) (ratelimit.Limiter, error) {
	o := make([]ratelimit.LimiterOption, 0, len(opts)+1)

	if c.WaitLimit > 0 {
		o = append(o, ratelimit.WaitLimit(c.WaitLimit))
	}

	o = append(o, opts...)

	switch c.Strategy {
	case "fixed-window":
		return ratelimit.NewFixedWindow(cache, name, c.Limit, c.Window, o...)
	case "token-bucket":
		return ratelimit.NewTokenBucket(cache, name, c.Rate, c.Burst, o...)
	default:
		return nil, errors.New("unsupported rate limiter strategy")
	}
}
