package config

import (
	"net/url"
	"os"
	"strings"

	"azugo.io/azugo/validation"
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
	origins := ""
	if origin := strings.Split(os.Getenv("CORS_ORIGINS"), ";"); len(origin) > 0 {
		if o, err := url.Parse(origin[0]); err == nil && len(o.Host) > 0 {
			origins = o.Host
		}
	}

	v.SetDefault(prefix+".origins", origins)
}
