package main

import (
	"context"
	"fmt"
	"log/slog"

	godbus "github.com/godbus/dbus/v5"
	"github.com/thomas-btst/hometrustd/dbus"
)

const (
	dbusNetworkManagerInterface        string = "org.freedesktop.NetworkManager"
	dbusFreedesktopPropertiesInterface string = "org.freedesktop.DBus.Properties"
)

const dbusNetworkManagerPath dbus.Path = "/org/freedesktop/NetworkManager"

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

type NetworkMonitor struct {
	client *dbus.Client
}

type WifiState struct {
	Connected bool
	SSID      string
	BSSID     string
}

func (w WifiState) String() string {
	if !w.Connected {
		return "Disconnected"
	}
	return fmt.Sprintf("Connected (SSID: %s, BSSID: %s)", w.SSID, w.BSSID)
}

func NewNetworkMonitor(conn *godbus.Conn) *NetworkMonitor {
	return &NetworkMonitor{
		client: dbus.NewClient(dbusNetworkManagerInterface, conn),
	}
}

func (n *NetworkMonitor) Watch(ctx context.Context) (<-chan WifiState, error) {
	conn := n.client.Conn

	matchOptions := []godbus.MatchOption{
		godbus.WithMatchObjectPath(godbus.ObjectPath(dbusNetworkManagerPath)),
		godbus.WithMatchInterface(dbusFreedesktopPropertiesInterface),
		godbus.WithMatchMember(dbusPropertiesSignal),
	}

	if err := conn.AddMatchSignal(matchOptions...); err != nil {
		return nil, fmt.Errorf("failed to add dbus properties match signal: %w", err)
	}

	signals := make(chan *godbus.Signal, 10)
	conn.Signal(signals)

	states := make(chan WifiState, cap(signals))

	go func() {
		defer close(states)
		defer conn.RemoveSignal(signals)
		defer func() {
			_ = conn.RemoveMatchSignal(matchOptions...)
		}()

		logError := func(err error) {
			slog.Error(
				"Error while getting network info",
				slog.Any("error", err),
			)
		}

		send := func(wifiState WifiState) bool {
			select {
			case states <- wifiState:
				return true
			case <-ctx.Done():
				return false
			}
		}

		state, err := n.Info()
		if err != nil {
			logError(err)
		}

		if !send(state) {
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-signals:
				if !ok {
					return
				}

				currentState, err := n.Info()
				if err != nil {
					logError(err)
				}

				if currentState == state {
					continue
				}

				state = currentState

				if !send(state) {
					return
				}
			}
		}
	}()

	return states, nil
}

func (n *NetworkMonitor) Info() (WifiState, error) {
	disconnected := WifiState{Connected: false}

	// 1. Primary active connection
	connPath, err := n.client.Object(dbusNetworkManagerPath).Path(dbusPrimaryProperty)
	switch {
	case err != nil:
		return disconnected, fmt.Errorf("failed to read primary connection: %w", err)
	case connPath.IsEmpty():
		return disconnected, nil
	}

	// 2. Network devices associated with active connection
	devPaths, err := n.client.Object(connPath).Paths(dbusDevicesProperty)
	if err != nil {
		return disconnected, fmt.Errorf("failed to read connection devices: %w", err)
	}
	if len(devPaths) == 0 {
		return disconnected, nil
	}

	devPath := dbus.EmptyPath
	for _, path := range devPaths {
		deviceType, err := n.client.Object(path).Uint32(dbusDeviceTypeProperty)
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
	apPath, err := n.client.Object(devPath).Path(dbusAccessPointProperty)
	switch {
	case err != nil:
		return disconnected, fmt.Errorf("failed to read active access point: %w", err)
	case apPath.IsEmpty():
		return disconnected, nil
	}

	// 4. Read SSID and HwAddress (BSSID) properties from AccessPoint D-Bus object
	appObject := n.client.Object(apPath)

	bssid, err := appObject.String(dbusBSSIDProperty)
	if err != nil {
		return disconnected, fmt.Errorf("failed to read access point HwAddress: %w", err)
	}

	ssidBytes, err := appObject.Bytes(dbusSSIDProperty)
	if err != nil {
		return disconnected, fmt.Errorf("failed to read access point Ssid: %w", err)
	}

	return WifiState{
		Connected: true,
		SSID:      string(ssidBytes),
		BSSID:     bssid,
	}, nil
}
