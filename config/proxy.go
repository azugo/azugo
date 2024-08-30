package config

import (
	"os"
	"strings"

	"azugo.io/core/validation"
	"github.com/spf13/viper"
)

// Proxy is a configuration for trusted proxies.
type Proxy struct {
	Address        []string `mapstructure:"address" validate:"dive,required,ip_addr|cidr|eq=*"`
	Limit          int      `mapstructure:"limit" validate:"min=0,max=10"`
	TrustedHeaders []string `mapstructure:"trusted_headers"`
}

// Validate Proxy configuration section.
func (c *Proxy) Validate(valid *validation.Validate) error {
	return valid.Struct(c)
}

// Bind Proxy configuration section.
func (c *Proxy) Bind(prefix string, v *viper.Viper) {
	addrs := []string{"127.0.0.1"}
	if env := os.Getenv("REVERSE_PROXY_TRUSTED_IPS"); len(env) > 0 {
		addrs = make([]string, 0, 3)

		for _, addr := range strings.Split(env, ";") {
			addr = strings.TrimSpace(addr)
			if len(addr) == 0 {
				continue
			}

			addrs = append(addrs, addr)
		}
	}

	headers := []string{"X-Real-IP", "X-Forwarded-For"}
	if env := os.Getenv("REVERSE_PROXY_TRUSTED_HEADERS"); len(env) > 0 {
		headers = make([]string, 0, 3)

		for _, header := range strings.Split(env, ";") {
			header = strings.TrimSpace(header)
			if len(header) == 0 {
				continue
			}

			headers = append(headers, header)
		}
	}

	v.SetDefault(prefix+".address", addrs)
	v.SetDefault(prefix+".limit", 1)
	v.SetDefault(prefix+".trusted_headers", headers)

	_ = v.BindEnv(prefix+".limit", "REVERSE_PROXY_LIMIT")
}
