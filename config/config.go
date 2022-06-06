package config

import (
	"errors"
	"fmt"

	"azugo.io/azugo/validation"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Configuration for the application.
type Configuration struct {
	v          *viper.Viper
	loaded     bool
	configName string
	validate   *validation.Validate

	// Server configuration section.
	Server *Server `mapstructure:"server"`
}

// New returns a new configuration.
func New() *Configuration {
	v := viper.New()
	v.AddConfigPath("./configs/")

	c := &Configuration{
		v:        v,
		validate: validation.New(),
	}

	return c
}

// Binder is an interface that can be implemented by configuration
// sections to bind to configuration file.
type Binder interface {
	Bind(prefix string, v *viper.Viper)
}

// Configurable is an interface that can be implemented by
// extended configuration.
type Configurable interface {
	Core() *Configuration
	Loaded(*Configuration)
}

// Validatable is an interface that can be implemented by configuration
// section to validate the configuration.
type Validatable interface {
	Validate(*validation.Validate) error
}

// Bind configuration section if it implements Binder interface.
func Bind[T any](c *T, prefix string, v *viper.Viper) *T {
	if c == nil {
		c = new(T)
	}
	if b, ok := any(c).(Binder); ok {
		b.Bind(prefix, v)
	}
	return c
}

// Bind binds configuration section to viper.
func (c *Configuration) Bind(_ string, v *viper.Viper) {
	c.Server = Bind(c.Server, "server", v)
}

// BindCmd adds configuration bindings from command arguments.
func (c *Configuration) BindCmd(cmd *cobra.Command) {
	// Special flags for the application.
	_ = c.v.BindPFlag("server.port", cmd.Flags().Lookup("port"))
}

// Core returns the core configuration.
func (c *Configuration) Core() *Configuration {
	return c
}

// Loaded receives loaded core configuration.
func (c *Configuration) Loaded(*Configuration) {}

// SetConfigFile explicitly defines the path, name and extension of the config file.
func (c *Configuration) SetConfigFile(path string) {
	c.v.SetConfigFile(path)
}

// SetConfigDirName sets the name of the directory where the config file is located under common config locations.
func (c *Configuration) SetConfigDirName(dirName string) {
	c.v.AddConfigPath("/etc/" + dirName + "/")
	c.v.AddConfigPath("$HOME/." + dirName + "/")
}

// SetConfigName sets name for the config file.
// Does not include extension.
func (c *Configuration) SetConfigName(name string) {
	c.configName = name
	c.v.SetConfigName(name)
}

// Load loads the configuration from the provided path.
func (c *Configuration) Load(config interface{}, environment string) error {
	if c.loaded {
		return nil
	}

	// Bind defaults
	extconf, ok := config.(Configurable)
	if !ok {
		return errors.New("configuration must implement Configurable interface")
	}

	conf := extconf.Core()
	conf.Bind("", c.v)

	if extbind, ok := config.(Binder); ok {
		extbind.Bind("", c.v)
	}

	// Load configuration
	configPath := c.v.ConfigFileUsed()

	if len(configPath) == 0 && len(c.configName) == 0 {
		c.SetConfigName("app")
	}

	if err := c.v.ReadInConfig(); err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return fmt.Errorf("failed to read configuration: %w", err)
		}
	}

	if len(configPath) == 0 && len(environment) > 0 {
		c.v.SetConfigName(c.configName + "." + environment)
		if err := c.v.MergeInConfig(); err != nil {
			if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
				return fmt.Errorf("failed to merge configuration: %w", err)
			}
		}
	}

	if err := c.v.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if err := c.Validate(c.validate); err != nil {
		return err
	}

	if extvalid, ok := config.(Validatable); ok {
		if err := extvalid.Validate(c.validate); err != nil {
			return err
		}
	}

	extconf.Loaded(c)
	c.loaded = true

	return nil
}

func (c *Configuration) Validate(validate *validation.Validate) error {
	if err := c.Server.Validate(validate); err != nil {
		return err
	}
	return nil
}

// Ready returns true if the configuration has been loaded.
func (c *Configuration) Ready() bool {
	return c.loaded
}
