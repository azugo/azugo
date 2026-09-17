package config

import (
	"time"

	"azugo.io/core/validation"
	"github.com/spf13/viper"
)

// Flash is the flash message configuration.
type Flash struct {
	// CookieName carries the opaque flash record ID between requests.
	CookieName string `mapstructure:"cookie_name" validate:"required"`
	// TTL bounds how long an unread flash record waits.
	TTL time.Duration `mapstructure:"ttl" validate:"omitempty,min=0"`
}

// Validate Flash configuration section.
func (c *Flash) Validate(valid *validation.Validate) error {
	return valid.Struct(c)
}

// Bind Flash configuration section.
func (c *Flash) Bind(prefix string, v *viper.Viper) {
	v.SetDefault(prefix+".cookie_name", "flash")
	v.SetDefault(prefix+".ttl", time.Duration(0))

	_ = v.BindEnv(prefix+".cookie_name", "FLASH_COOKIE_NAME")
	_ = v.BindEnv(prefix+".ttl", "FLASH_TTL")
}
