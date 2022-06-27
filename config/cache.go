package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"time"

	"azugo.io/azugo/cache"
	"azugo.io/azugo/validation"

	"github.com/spf13/viper"
)

type Cache struct {
	Type             cache.CacheType `mapstructure:"type" validate:"required,oneof=memory redis"`
	TTL              time.Duration   `mapstructure:"ttl" validate:"omitempty,min=0"`
	ConnectionString string          `mapstructure:"connection" validate:"omitempty"`
	Password         string          `mapstructure:"password" validate:"omitempty"`
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
	var psw string
	passpath := os.Getenv("CACHE_PASSWORD_FILE")
	if _, err := os.Stat(passpath); err == nil {
		if content, err := ioutil.ReadFile(passpath); err == nil && len(content) > 0 {
			psw = string(bytes.TrimSpace(content))
		}
	}

	v.SetDefault(prefix+".type", "memory")
	v.SetDefault(prefix+".password", psw)

	_ = v.BindEnv(prefix+".type", "CACHE_TYPE")
	_ = v.BindEnv(prefix+".ttl", "CACHE_TTL")
	_ = v.BindEnv(prefix+".connection", "CACHE_CONNECTION")
	_ = v.BindEnv(prefix+".key_prefix", "CACHE_KEY_PREFIX")
}
