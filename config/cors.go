package config

import (
	"net/url"
	"os"
	"strings"

	"azugo.io/core/validation"
	"github.com/spf13/viper"
)

// CORS is a configuration for CORS middleware.
type CORS struct {
	Origins []string `mapstructure:"origins" validate:"dive,required,url"`
}

// Validate CORS configuration section.
func (c *CORS) Validate(valid *validation.Validate) error {
	return valid.Struct(c)
}

// Bind CORS configuration section.
func (c *CORS) Bind(prefix string, v *viper.Viper) {
	origins := make([]string, 0, 1)

	if origin := strings.Split(os.Getenv("CORS_ORIGINS"), ";"); len(origin) > 0 {
		for _, addr := range origin {
			if len(addr) == 0 {
				continue
			}

			if o, err := url.Parse(strings.TrimSpace(addr)); err == nil && len(o.Host) > 0 {
				origins = append(origins, o.Scheme+"://"+o.Host)
			}
		}
	}

	v.SetDefault(prefix+".origins", origins)
}
