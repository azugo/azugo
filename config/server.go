package config

import (
	"net/url"
	"os"
	"strconv"
	"strings"

	"azugo.io/core/config"
	"azugo.io/core/validation"
	"github.com/spf13/viper"
)

// ServerHTTP is a HTTP server configuration.
type ServerHTTP struct {
	Enabled bool   `mapstructure:"enabled"`
	Address string `mapstructure:"address" validate:"ip_addr|hostname|fqdn"`
	Port    int    `mapstructure:"port" validate:"required,min=1,max=65535"`
}

// Bind server configuration section.
func (s *ServerHTTP) Bind(prefix string, v *viper.Viper) {
	// Special functionality for SERVER_URLS defaults
	addr := "0.0.0.0"
	port := 80
	enabled := true

	for _, servu := range strings.Split(os.Getenv("SERVER_URLS"), ";") {
		servu = strings.TrimSpace(servu)
		if len(servu) == 0 {
			continue
		}
		if u, err := url.Parse(servu); err == nil {
			if strings.ToLower(u.Scheme) != "http" {
				enabled = false
				continue
			}
			enabled = true
			addr = u.Hostname()
			if p, err := strconv.Atoi(u.Port()); err == nil {
				port = p
			}
			break
		}
	}

	v.SetDefault(prefix+".enabled", enabled)
	v.SetDefault(prefix+".address", addr)
	v.SetDefault(prefix+".port", port)
}

// Validate server configuration section.
func (s *ServerHTTP) Validate(valid *validation.Validate) error {
	return valid.Struct(s)
}

// ServerHTTPS is a HTTPS server configuration.
type ServerHTTPS struct {
	Enabled            bool   `mapstructure:"enabled"`
	Address            string `mapstructure:"address" validate:"ip_addr|hostname|fqdn"`
	Port               int    `mapstructure:"port" validate:"required,min=1,max=65535"`
	CertificatePEMFile string `mapstructure:"certificate_pem_file" validate:"omitempty,file"`
}

// Bind server configuration section.
func (s *ServerHTTPS) Bind(prefix string, v *viper.Viper) {
	// Special functionality for SERVER_URLS defaults
	addr := "0.0.0.0"
	port := 443
	enabled := false

	for _, servu := range strings.Split(os.Getenv("SERVER_URLS"), ";") {
		servu = strings.TrimSpace(servu)
		if len(servu) == 0 {
			continue
		}
		if u, err := url.Parse(servu); err == nil {
			if strings.ToLower(u.Scheme) != "https" {
				continue
			}
			enabled = true
			addr = u.Hostname()
			if p, err := strconv.Atoi(u.Port()); err == nil {
				port = p
			}
			break
		}
	}

	v.SetDefault(prefix+".enabled", enabled)
	v.SetDefault(prefix+".address", addr)
	v.SetDefault(prefix+".port", port)

	_ = v.BindEnv(prefix+".certificate_pem_file", "SERVER_HTTPS_CERTIFICATE_PEM_FILE")
}

// Validate server configuration section.
func (s *ServerHTTPS) Validate(valid *validation.Validate) error {
	return valid.Struct(s)
}

// Server configuration section.
type Server struct {
	HTTP  *ServerHTTP  `mapstructure:"http"`
	HTTPS *ServerHTTPS `mapstructure:"https"`
	Path  string       `mapstructure:"path"`
}

// Bind server configuration section.
func (s *Server) Bind(prefix string, v *viper.Viper) {
	// Special functionality for SERVER_URLS defaults
	path := "/"

	if servu := strings.Split(os.Getenv("SERVER_URLS"), ";"); len(servu) > 0 && len(servu[0]) > 0 {
		if u, err := url.Parse(servu[0]); err == nil && len(u.Path) > 0 {
			path = u.Path
		}
	}

	v.SetDefault(prefix+".path", path)

	_ = v.BindEnv(prefix+".path", "BASE_PATH")

	s.HTTP = config.Bind(s.HTTP, prefix+".http", v)
	s.HTTPS = config.Bind(s.HTTPS, prefix+".https", v)
}

// Validate server configuration section.
func (s *Server) Validate(valid *validation.Validate) error {
	if s.HTTP != nil {
		if err := s.HTTP.Validate(valid); err != nil {
			return err
		}
	}
	if s.HTTPS != nil {
		if err := s.HTTPS.Validate(valid); err != nil {
			return err
		}
	}
	return valid.Struct(s)
}
