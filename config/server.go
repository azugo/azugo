package config

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

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
	port := 8080
	enabled := true

	for servu := range strings.SplitSeq(os.Getenv("SERVER_URLS"), ";") {
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
	port := 4443
	enabled := false

	for servu := range strings.SplitSeq(os.Getenv("SERVER_URLS"), ";") {
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

	// Maximum duration for reading the full request including body.
	ReadTimeout time.Duration `mapstructure:"read_timeout" validate:"omitempty,min=0"`
	// Maximum duration for writing the response.
	WriteTimeout time.Duration `mapstructure:"write_timeout" validate:"omitempty,min=0"`
	// Maximum duration to wait for the next request on a keep-alive connection.
	IdleTimeout time.Duration `mapstructure:"idle_timeout" validate:"omitempty,min=0"`
	// Maximum request body size.
	MaxRequestBodySize int `mapstructure:"max_request_body_size" validate:"omitempty,min=0"`
	// Maximum duration to wait for active connections to finish on shutdown.
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"omitempty,min=0"`
}

// Bind server configuration section.
func (s *Server) Bind(prefix string, v *viper.Viper) {
	// Special functionality for SERVER_URLS defaults
	path := "/"

	if servu, _, _ := strings.Cut(os.Getenv("SERVER_URLS"), ";"); len(servu) > 0 {
		if u, err := url.Parse(servu); err == nil && len(u.Path) > 0 {
			path = u.Path
		}
	}

	v.SetDefault(prefix+".path", path)
	v.SetDefault(prefix+".read_timeout", 30*time.Second)
	v.SetDefault(prefix+".write_timeout", 10*time.Second)
	v.SetDefault(prefix+".idle_timeout", 75*time.Second)
	v.SetDefault(prefix+".max_request_body_size", 4<<20)
	v.SetDefault(prefix+".shutdown_timeout", 30*time.Second)

	_ = v.BindEnv(prefix+".path", "BASE_PATH")
	_ = v.BindEnv(prefix+".read_timeout", "SERVER_READ_TIMEOUT")
	_ = v.BindEnv(prefix+".write_timeout", "SERVER_WRITE_TIMEOUT")
	_ = v.BindEnv(prefix+".idle_timeout", "SERVER_IDLE_TIMEOUT")
	_ = v.BindEnv(prefix+".max_request_body_size", "SERVER_MAX_REQUEST_BODY_SIZE")
	_ = v.BindEnv(prefix+".shutdown_timeout", "SERVER_SHUTDOWN_TIMEOUT")

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
