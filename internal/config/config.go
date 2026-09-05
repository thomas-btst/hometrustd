// Package config provides configuration structures and validation for the application.
package config

type Config struct {
	TrustedNetworks TrustedNetworks `mapstructure:"trusted_networks"`
}

func (c *Config) Validate() error {
	return c.TrustedNetworks.Validate()
}
