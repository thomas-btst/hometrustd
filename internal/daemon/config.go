package daemon

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

type Config struct {
	trustedNetworks map[string]struct{}
}

func NewConfig(trustedNetworks []string) *Config {
	trustNetMap := make(map[string]struct{}, len(trustedNetworks))
	for _, bssid := range trustedNetworks {
		trustNetMap[strings.ToUpper(bssid)] = struct{}{}
	}

	return &Config{
		trustedNetworks: trustNetMap,
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

func (c *Config) IsTrustedNetwork(bssid string) bool {
	if c == nil || len(c.trustedNetworks) == 0 {
		return false
	}

	_, ok := c.trustedNetworks[strings.ToUpper(bssid)]
	return ok
}
