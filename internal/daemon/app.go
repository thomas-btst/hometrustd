// Package daemon provides the main entry point for the hometrustd daemon, which monitors network connectivity and manages idle inhibition based on Network
package daemon

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/thomas-btst/hometrustd/internal/config"
	"github.com/thomas-btst/hometrustd/internal/network"
)

const ProgramName = "hometrustd"

type NetworkWatcher interface {
	Watch(ctx context.Context) (<-chan struct{}, error)
	State() network.State
}

type IdleInhibitor interface {
	Inhibit(reason string) error
	Uninhibit() error
}

type App struct {
	networkWatcher NetworkWatcher
	idleInhibitor  IdleInhibitor
	configStore    *config.Store
}

func NewApp(networkWatcher NetworkWatcher, idleInhibitor IdleInhibitor, cfgStore *config.Store) *App {
	return &App{
		networkWatcher: networkWatcher,
		idleInhibitor:  idleInhibitor,
		configStore:    cfgStore,
	}
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		if err := a.idleInhibitor.Uninhibit(); err != nil {
			slog.Error("Failed to uninhibit idle on exit", slog.Any("error", err))
		}
	}()

	netEvents, err := a.networkWatcher.Watch(ctx)
	if err != nil {
		return fmt.Errorf("failed to start network monitor watch: %w", err)
	}

	cfgEvents := a.configStore.Watch(ctx)

	a.updateIdleInhibition()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-netEvents:
			a.updateIdleInhibition()
		case <-cfgEvents:
			a.updateIdleInhibition()
		}
	}
}

func (a *App) updateIdleInhibition() {
	netState := a.networkWatcher.State()
	trustNets := a.configStore.Current().TrustedNetworks

	alias, ok := trustNets.BSSIDs[netState.BSSID]
	if !netState.Connected || !ok {
		if netState.Connected {
			slog.Info(
				"Connected to untrusted Wi-Fi network",
				slog.String("bssid", string(netState.BSSID)),
				slog.String("ssid", netState.SSID),
			)
		} else {
			slog.Info("Disconnected from Wi-Fi network")
		}

		if err := a.idleInhibitor.Uninhibit(); err != nil {
			slog.Error("Failed to uninhibit idle", slog.Any("error", err))
		}
		return
	}

	slog.Info(
		"Connected to trusted Wi-Fi network",
		stringAttr("alias", alias),
		slog.String("bssid", string(netState.BSSID)),
		slog.String("ssid", netState.SSID),
	)

	reason := fmt.Sprintf("Connected to trusted Wi-Fi network %s (%s)", netState.SSID, netState.BSSID)
	if alias != "" {
		reason = fmt.Sprintf("Connected to trusted Wi-Fi network '%s' [%s] (%s)", alias, netState.SSID, netState.BSSID)
	}

	if err := a.idleInhibitor.Inhibit(reason); err != nil {
		slog.Error("Failed to inhibit idle", slog.Any("error", err))
	}
}

func stringAttr(key, val string) slog.Attr {
	if val == "" {
		return slog.Attr{}
	}
	return slog.String(key, val)
}
