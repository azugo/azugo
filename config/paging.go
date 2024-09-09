package config

import (
	"azugo.io/core/paginator"
	"azugo.io/core/validation"
	"github.com/spf13/viper"
)

type Paging struct {
	// DefaultPageSize represents the default number of items per page.
	DefaultPageSize int `mapstructure:"default_page_size" validate:"required,min=1"`
	// MaxPageSize represents the default maximum number of items per page.
	MaxPageSize int `mapstructure:"max_page_size" validate:"required,min=1"`
}

// Validate Paging configuration section.
func (c *Paging) Validate(valid *validation.Validate) error {
	return valid.Struct(c)
}

// Bind Paging configuration section.
func (c *Paging) Bind(prefix string, v *viper.Viper) {
	v.SetDefault(prefix+".default_page_size", paginator.DefaultPageSize)
	v.SetDefault(prefix+".max_page_size", 100)

	_ = v.BindEnv(prefix+".default_page_size", "PAGING_DEFAULT_PAGE_SIZE")
	_ = v.BindEnv(prefix+".max_page_size", "PAGING_MAX_PAGE_SIZE")
}
