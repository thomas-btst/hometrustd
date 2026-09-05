package config

import (
	"errors"
	"fmt"
	"net"

	"github.com/thomas-btst/hometrustd/internal/network"
)

type TrustedNetworks struct {
	BSSIDs map[network.BSSID]string `mapstructure:"bssids"`
}

func (c *TrustedNetworks) Validate() error {
	if c == nil || len(c.BSSIDs) == 0 {
		return errors.New("no trusted networks configured, daemon will not inhibit idle state for any network")
	}

	var errs []error
	for bssid := range c.BSSIDs {
		strBSSID := string(bssid)
		if _, err := net.ParseMAC(strBSSID); err != nil {
			errs = append(errs, fmt.Errorf("invalid BSSID format in trusted networks '%s'", bssid))
		}
	}

	return errors.Join(errs...)
}
