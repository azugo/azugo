package azugo

import (
	"os"
)

// Environment type.
type Environment string

const (
	EnvironmentDevelopment Environment = "Development"
	EnvironmentStaging                 = "Staging"
	EnvironmentProduction              = "Production"
)

// NewEnvironment creates new Environment instance.
func NewEnvironment(defaultMode string) Environment {
	env := os.Getenv("ENVIRONMENT")
	if len(env) == 0 {
		env = defaultMode
	}

	if env == EnvironmentProduction || env == EnvironmentStaging {
		return Environment(env)
	}

	return EnvironmentDevelopment
}

// IsProduction checks if current environment is production.
func (e Environment) IsProduction() bool {
	return e == EnvironmentProduction
}

// IsStaging checks if current environment is staging.
func (e Environment) IsStaging() bool {
	return e == EnvironmentStaging
}

// IsDevelopment checks if current environment is development.
func (e Environment) IsDevelopment() bool {
	return e == EnvironmentDevelopment
}
