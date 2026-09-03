package daemon

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

type Config struct {
	trustedNetworks map[string]string
}

func NewConfig(trustedNetworks map[string]string) *Config {
	normalized := make(map[string]string, len(trustedNetworks))
	for bssid, alias := range trustedNetworks {
		normalized[strings.ToUpper(bssid)] = alias
	}
	return &Config{
		trustedNetworks: normalized,
	}
}

func (c *Config) Validate() error {
	if c == nil || len(c.trustedNetworks) == 0 {
		return errors.New("no trusted networks configured. Daemon will not inhibit idle state for any network")
	}

	for bssid := range c.trustedNetworks {
		if _, err := net.ParseMAC(bssid); err != nil {
			return fmt.Errorf("invalid BSSID format in trusted networks '%s'", bssid)
		}
	}

	return nil
}

func (c *Config) Lookup(bssid string) (string, bool) {
	if c == nil || len(c.trustedNetworks) == 0 {
		return "", false
	}

	alias, ok := c.trustedNetworks[strings.ToUpper(bssid)]
	return alias, ok
}
