package config

import (
	"time"

	"azugo.io/azugo/cache"
	"azugo.io/azugo/validation"

	"github.com/spf13/viper"
)

type Cache struct {
	Type             cache.CacheType `mapstructure:"type" validate:"required,oneof=memory redis"`
	TTL              time.Duration   `mapstructure:"ttl" validate:"omitempty,min=0"`
	ConnectionString string          `mapstructure:"connection" validate:"omitempty"`
	KeyPrefix        string          `mapstructure:"key_prefix" validate:"omitempty"`
}

// Validate cache configuration section.
func (c *Cache) Validate(valid *validation.Validate) error {
	if err := valid.Struct(c); err != nil {
		return err
	}
	if err := cache.ValidateConnectionString(c.Type, c.ConnectionString); err != nil {
		return err
	}
	return nil
}

// Bind cache configuration section.
func (c *Cache) Bind(prefix string, v *viper.Viper) {
	v.SetDefault(prefix+".type", "memory")
}
