package config

import (
	"azugo.io/core/config"
	"azugo.io/core/validation"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Configuration for the application.
type Configuration struct {
	*config.Configuration `mapstructure:",squash"`

	// Server configuration section.
	Server *Server `mapstructure:"server"`
	// CORS configuration section.
	CORS *CORS `mapstructure:"cors"`
	// Proxy configuration section.
	Proxy *Proxy `mapstructure:"proxy"`
	// Metrics configuration section.
	Metrics *Metrics `mapstructure:"metrics"`
}

// New returns a new configuration.
func New() *Configuration {
	return &Configuration{
		Configuration: config.New(),
	}
}

// Configurable is an interface that can be implemented by
// extended configuration.
type Configurable interface {
	ServerCore() *Configuration
}

// Bind configuration section if it implements Binder interface.
func Bind[T any](c *T, prefix string, v *viper.Viper) *T {
	return config.Bind(c, prefix, v)
}

// Bind binds configuration section to viper.
func (c *Configuration) Bind(_ string, v *viper.Viper) {
	c.Server = config.Bind(c.Server, "server", v)
	c.CORS = config.Bind(c.CORS, "cors", v)
	c.Proxy = config.Bind(c.Proxy, "proxy", v)
	c.Metrics = config.Bind(c.Metrics, "metrics", v)
}

// BindCmd adds configuration bindings from command arguments.
func (c *Configuration) BindCmd(cmd *cobra.Command, v *viper.Viper) {
	// Special flags for the application.
	_ = v.BindPFlag("server.port", cmd.Flags().Lookup("port"))
}

// ServerCore returns the web core configuration.
func (c *Configuration) ServerCore() *Configuration {
	return c
}

// Validate the configuration.
func (c *Configuration) Validate(validate *validation.Validate) error {
	if err := c.Server.Validate(validate); err != nil {
		return err
	}
	if err := c.CORS.Validate(validate); err != nil {
		return err
	}
	if err := c.Proxy.Validate(validate); err != nil {
		return err
	}
	if err := c.Metrics.Validate(validate); err != nil {
		return err
	}
	return nil
}
