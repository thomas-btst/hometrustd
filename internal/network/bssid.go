package network

import (
	"strings"
)

type BSSID string

func (b BSSID) String() string {
	return string(b)
}

func NewBSSID(bssid string) BSSID {
	normalizedBSSID := strings.ToLower(bssid)
	return BSSID(normalizedBSSID)
}

func (b *BSSID) UnmarshalText(text []byte) error {
	bssid := NewBSSID(string(text))
	*b = BSSID(bssid)
	return nil
}
