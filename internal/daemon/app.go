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
}

func NewApp(networkWatcher NetworkWatcher, idleInhibitor IdleInhibitor) *App {
	return &App{
		networkWatcher: networkWatcher,
		idleInhibitor:  idleInhibitor,
	}
}

func (a *App) Run(ctx context.Context) error {
	networkStates, err := a.networkWatcher.Watch(ctx)
	if err != nil {
		return fmt.Errorf("failed to start network monitor watch: %w", err)
	}

	for networkState := range networkStates {
		slog.Info("Network state", slog.Any("state", networkState)) // TODO: move to connected to home

		if networkState.Connected {
			if err := a.idleInhibitor.Inhibit(
				fmt.Sprintf(
					"Connected to Wi-Fi network %s (%s)",
					networkState.SSID, networkState.BSSID,
				),
			); err != nil {
				slog.Error("Failed to inhibit idle", slog.Any("error", err))
			}
		} else {
			if err := a.idleInhibitor.Uninhibit(); err != nil {
				slog.Error("Failed to uninhibit idle", slog.Any("error", err))
			}
		}
	}

	return nil
}
