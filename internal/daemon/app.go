// Package daemon provides the main entry point for the hometrustd daemon, which monitors network connectivity and manages idle inhibition based on Network
package daemon

import (
	"context"
	"fmt"
	"log/slog"
)

type TrustedWatcher interface {
	Watch(ctx context.Context) (<-chan struct{}, error)
	State() TrustState
}

type IdleInhibitor interface {
	Inhibit(reason string) error
	Uninhibit() error
	Start(ctx context.Context) error
}

type NotifySender interface {
	Send(summary, body string) error
}

type App struct {
	trustedWatcher TrustedWatcher
	idleInhibitor  IdleInhibitor
	notifySender   NotifySender
}

func NewApp(trustedWatcher TrustedWatcher, idleInhibitor IdleInhibitor, notifySender NotifySender) *App {
	return &App{
		trustedWatcher: trustedWatcher,
		idleInhibitor:  idleInhibitor,
		notifySender:   notifySender,
	}
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		if err := a.idleInhibitor.Uninhibit(); err != nil {
			slog.Error("Failed to uninhibit idle on exit", slog.Any("error", err))
		}
	}()

	if err := a.idleInhibitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to run idle inhibitor: %w", err)
	}

	events, err := a.trustedWatcher.Watch(ctx)
	if err != nil {
		return fmt.Errorf("failed to start trusted monitor: %w", err)
	}

	a.applyState(true)
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-events:
			if !ok {
				return nil
			}
			a.applyState(false)
		}
	}
}

func (a *App) applyState(isInitial bool) {
	trustState := a.trustedWatcher.State()

	if !trustState.Trusted {
		if isInitial {
			return
		}

		slog.Info("Untrusted network state, system idle behaviors restored")

		if err := a.idleInhibitor.Uninhibit(); err != nil {
			slog.Error("Failed to uninhibit idle", slog.Any("error", err))
		}

		err := a.notifySender.Send("Disconnected from trusted Wi-Fi", "System idle behaviors restored")
		if err != nil {
			slog.Error("Failed to send notification", slog.Any("error", err))
		}

		return
	}

	slog.Info("Trusted Wi-Fi network active, system idle inhibition enabled", slog.String("network", trustState.Name))

	if err := a.notifySender.Send(
		fmt.Sprintf("Connected to Wi-Fi %s", trustState.Name),
		"System idle behaviors disabled",
	); err != nil {
		slog.Error("Failed to send notification", slog.Any("error", err))
	}

	reason := fmt.Sprintf("Connected to trusted Wi-Fi network %s", trustState.Name)

	if err := a.idleInhibitor.Inhibit(reason); err != nil {
		slog.Error("Failed to inhibit idle", slog.Any("error", err))
	}
}
