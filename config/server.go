package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"

	"azugo.io/azugo/validation"
	"github.com/spf13/viper"
)

// Server configuration section.
type Server struct {
	Address string `mapstructure:"address" validate:"ip4_addr|ip6_addr|hostname|fqdn"`
	Port    int    `mapstructure:"port" validate:"required,min=1,max=65535"`
	Path    string `mapstructure:"path"`
}

// Bind server configuration section.
func (s *Server) Bind(prefix string, v *viper.Viper) {
	// Special functionality for SERVER_URL defaults
	addr := "0.0.0.0"
	port := 80
	path := "/"

	if servu := os.Getenv("SERVER_URL"); len(servu) != 0 {
		if u, err := url.Parse(servu); err == nil {
			addr = u.Hostname()
			if p, err := strconv.Atoi(u.Port()); err == nil {
				port = p
			}
			path = u.Path
		}
	}

	v.SetDefault(fmt.Sprintf("%s.address", prefix), addr)
	v.SetDefault(fmt.Sprintf("%s.port", prefix), port)
	v.SetDefault(fmt.Sprintf("%s.path", prefix), path)

	_ = v.BindEnv(fmt.Sprintf("%s.path", prefix), "BASE_PATH")
}

// Validate server configuration section.
func (s *Server) Validate(valid *validation.Validate) error {
	return valid.Struct(s)
}
