// Package network provides a NetworkMonitor that uses D-Bus to monitor the Wi-Fi connection state on Linux systems using NetworkManager.
package network

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	godbus "github.com/godbus/dbus/v5"
	"github.com/thomas-btst/hometrustd/internal/dbus"
)

const (
	dbusNMInterface                    string = "org.freedesktop.NetworkManager"
	dbusFreedesktopPropertiesInterface string = "org.freedesktop.DBus.Properties"
)

const dbusNMPath dbus.Path = "/org/freedesktop/NetworkManager"

const (
	dbusPrimaryProperty     dbus.Property = "PrimaryConnection"
	dbusDevicesProperty     dbus.Property = "Connection.Active.Devices"
	dbusDeviceTypeProperty  dbus.Property = "Device.DeviceType"
	dbusAccessPointProperty dbus.Property = "Device.Wireless.ActiveAccessPoint"
	dbusSSIDProperty        dbus.Property = "AccessPoint.Ssid"
	dbusBSSIDProperty       dbus.Property = "AccessPoint.HwAddress"
)

const (
	dbusPropertiesSignal = "PropertiesChanged"
	dbusDeviceTypeWifi   = 2
)

type Monitor struct {
	client *dbus.Client
	mu     sync.Mutex
	state  State
}

type State struct {
	Connected bool
	BSSID     BSSID
	SSID      string
}

func NewMonitor(conn *godbus.Conn) *Monitor {
	return &Monitor{
		client: dbus.NewClient(dbusNMInterface, conn),
	}
}

func (m *Monitor) Watch(ctx context.Context) (<-chan struct{}, error) {
	matchOptions := []godbus.MatchOption{
		godbus.WithMatchObjectPath(godbus.ObjectPath(dbusNMPath)),
		godbus.WithMatchInterface(dbusFreedesktopPropertiesInterface),
		godbus.WithMatchMember(dbusPropertiesSignal),
	}

	signals, err := m.client.Signals(ctx, matchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dbus signals: %w", err)
	}

	netSignals := make(chan struct{}, 1)

	state, err := m.info()
	if err != nil {
		slog.Error(
			"Error while getting network info",
			slog.Any("error", err),
		)
	}

	m.mu.Lock()
	m.state = state
	m.mu.Unlock()

	go func() {
		defer close(netSignals)

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-signals:
				if !ok {
					return
				}

				newState, err := m.info()
				if err != nil {
					slog.Error(
						"Error while getting network info",
						slog.Any("error", err),
					)
				}

				m.mu.Lock()
				if newState == m.state {
					m.mu.Unlock()
					continue
				}

				m.state = newState
				m.mu.Unlock()

				select {
				case netSignals <- struct{}{}:
				default:
				}
			}
		}
	}()

	return netSignals, nil
}

func (m *Monitor) State() State {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.state
}

func (m *Monitor) info() (State, error) {
	disconnected := State{Connected: false}

	// 1. Primary active connection
	connPath, err := m.client.Object(dbusNMPath).Path(dbusPrimaryProperty)
	switch {
	case err != nil:
		return disconnected, fmt.Errorf("failed to read primary connection: %w", err)
	case connPath.IsEmpty():
		return disconnected, nil
	}

	// 2. Network devices associated with active connection
	devPaths, err := m.client.Object(connPath).Paths(dbusDevicesProperty)
	if err != nil {
		return disconnected, fmt.Errorf("failed to read connection devices: %w", err)
	}
	if len(devPaths) == 0 {
		return disconnected, nil
	}

	devPath := dbus.EmptyPath
	for _, path := range devPaths {
		deviceType, err := m.client.Object(path).Uint32(dbusDeviceTypeProperty)
		if err != nil {
			return disconnected, fmt.Errorf("failed to read device type: %w", err)
		}

		if deviceType == dbusDeviceTypeWifi {
			devPath = path
			break
		}
	}

	if devPath.IsEmpty() {
		return disconnected, nil
	}

	// 3. Active access point for Wi-Fi card
	apPath, err := m.client.Object(devPath).Path(dbusAccessPointProperty)
	switch {
	case err != nil:
		return disconnected, fmt.Errorf("failed to read active access point: %w", err)
	case apPath.IsEmpty():
		return disconnected, nil
	}

	// 4. Read SSID and HwAddress (BSSID) properties from AccessPoint D-Bus object
	appObject := m.client.Object(apPath)

	bssid, err := appObject.String(dbusBSSIDProperty)
	if err != nil {
		return disconnected, fmt.Errorf("failed to read access point HwAddress: %w", err)
	}

	ssidBytes, err := appObject.Bytes(dbusSSIDProperty)
	if err != nil {
		return disconnected, fmt.Errorf("failed to read access point Ssid: %w", err)
	}

	return State{
		Connected: true,
		BSSID:     NewBSSID(bssid),
		SSID:      string(ssidBytes),
	}, nil
}
