package config

import (
	"os"
	"strings"

	"azugo.io/core/validation"
	"github.com/spf13/viper"
)

// Healthz is the health check endpoint configuration.
type Healthz struct {
	Enabled bool     `mapstructure:"enabled"`
	Address []string `mapstructure:"address" validate:"dive,required,ip_addr|cidr|eq=*"`
}

// Validate Healthz configuration section.
func (c *Healthz) Validate(valid *validation.Validate) error {
	if !c.Enabled {
		return nil
	}

	return valid.Struct(c)
}

// Bind Healthz configuration section.
func (c *Healthz) Bind(prefix string, v *viper.Viper) {
	addrs := []string{
		"127.0.0.0/8",
		"::1/128",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7",
	}

	if env := os.Getenv("HEALTHZ_TRUSTED_IPS"); len(env) > 0 {
		addrs = make([]string, 0, 6)

		for _, addr := range strings.Split(env, ";") {
			addr = strings.TrimSpace(addr)
			if len(addr) == 0 {
				continue
			}

			addrs = append(addrs, addr)
		}
	}

	v.SetDefault(prefix+".enabled", true)
	v.SetDefault(prefix+".address", addrs)

	_ = v.BindEnv(prefix+".enabled", "HEALTHZ_ENABLED")
}
