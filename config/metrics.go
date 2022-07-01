package config

import (
	"os"
	"strings"

	"azugo.io/azugo/validation"
	"github.com/spf13/viper"
)

type Metrics struct {
	Enabled   bool     `mapstructure:"enabled"`
	Path      string   `mapstructure:"path"`
	Address   []string `mapstructure:"address" validate:"dive,required,ip_addr|cidr|eq=*"`
	SkipPaths []string `mapstructure:"skip_paths"`
}

// Validate Metrics configuration section.
func (c *Metrics) Validate(valid *validation.Validate) error {
	if !c.Enabled {
		return nil
	}
	return valid.Struct(c)
}

// Bind Metrics configuration section.
func (c *Metrics) Bind(prefix string, v *viper.Viper) {
	addrs := []string{"127.0.0.1"}
	if env := os.Getenv("METRICS_TRUSTED_IPS"); len(env) > 0 {
		addrs = make([]string, 0, 3)
		for _, addr := range strings.Split(env, ";") {
			addr = strings.TrimSpace(addr)
			if len(addr) == 0 {
				continue
			}
			addrs = append(addrs, addr)
		}
	}

	v.SetDefault(prefix+".enabled", true)
	v.SetDefault(prefix+".path", "/metrics")
	v.SetDefault(prefix+".address", addrs)

	_ = v.BindEnv(prefix+".enabled", "METRICS_ENABLED")
	_ = v.BindEnv(prefix+".path", "METRICS_PATH")
}
