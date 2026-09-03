// Package daemon provides the main entry point for the hometrustd daemon, which monitors network connectivity and manages idle inhibition based on Network
package daemon

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/thomas-btst/hometrustd/internal/network"
)

const ProgramName = "hometrustd"

type NetworkWatcher interface {
	Watch(ctx context.Context) (<-chan network.State, error)
}

type IdleInhibitor interface {
	Inhibit(reason string) error
	Uninhibit() error
}

type App struct {
	networkWatcher NetworkWatcher
	idleInhibitor  IdleInhibitor
	config         *Config
}

func NewApp(networkWatcher NetworkWatcher, idleInhibitor IdleInhibitor, cfg *Config) *App {
	return &App{
		networkWatcher: networkWatcher,
		idleInhibitor:  idleInhibitor,
		config:         cfg,
	}
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		if err := a.idleInhibitor.Uninhibit(); err != nil {
			slog.Error("Failed to uninhibit idle on exit", slog.Any("error", err))
		}
	}()

	networkStates, err := a.networkWatcher.Watch(ctx)
	if err != nil {
		return fmt.Errorf("failed to start network monitor watch: %w", err)
	}

	for networkState := range networkStates {
		alias, ok := a.config.Lookup(networkState.BSSID)
		if !networkState.Connected || !ok {
			if networkState.Connected {
				slog.Info(
					"Connected to untrusted Wi-Fi network",
					slog.String("bssid", networkState.BSSID),
					slog.String("ssid", networkState.SSID),
				)
			} else {
				slog.Info("Disconnected from Wi-Fi network")
			}

			if err := a.idleInhibitor.Uninhibit(); err != nil {
				slog.Error("Failed to uninhibit idle", slog.Any("error", err))
			}
			continue
		}

		slog.Info(
			"Connected to trusted Wi-Fi network",
			stringAttr("alias", alias),
			slog.String("bssid", networkState.BSSID),
			slog.String("ssid", networkState.SSID),
		)

		reason := fmt.Sprintf("Connected to trusted Wi-Fi network %s (%s)", networkState.SSID, networkState.BSSID)
		if alias != "" {
			reason = fmt.Sprintf("Connected to trusted Wi-Fi network '%s' [%s] (%s)", alias, networkState.SSID, networkState.BSSID)
		}

		if err := a.idleInhibitor.Inhibit(reason); err != nil {
			slog.Error("Failed to inhibit idle", slog.Any("error", err))
		}
	}

	return nil
}

func stringAttr(key, val string) slog.Attr {
	if val == "" {
		return slog.Attr{}
	}
	return slog.String(key, val)
}
