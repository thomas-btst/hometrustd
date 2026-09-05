// Package idle provides functionality to inhibit the system's idle state, preventing the screensaver or power management from activating while certain operations are ongoing.
package idle

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	godbus "github.com/godbus/dbus/v5"
	"github.com/thomas-btst/hometrustd/internal/daemon"
	"github.com/thomas-btst/hometrustd/internal/dbus"
)

const dbusScreensaverInterface string = "org.freedesktop.ScreenSaver"

const dbusScreensaverPath dbus.Path = "/org/freedesktop/ScreenSaver"

const (
	dbusInhibitCall   dbus.Method = "Inhibit"
	dbusUninhibitCall dbus.Method = "UnInhibit"
)

type inhibitionState struct {
	reason string
	cookie *uint32
}

type Inhibitor struct {
	client  *dbus.Client
	monitor *Monitor
	mu      sync.Mutex
	state   *inhibitionState
}

func NewInhibitor(conn *godbus.Conn, monitor *Monitor) *Inhibitor {
	return &Inhibitor{
		client:  dbus.NewClient(dbusScreensaverInterface, conn),
		monitor: monitor,
	}
}

func (i *Inhibitor) Start(ctx context.Context) error {
	idleEvent, err := i.monitor.Watch(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to idle monitor: %w", err)
	}

	if available := i.monitor.IsAvailable(); !available {
		slog.Warn("Screensaver service is not available, skipping inhibition...")
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-idleEvent:
				if !ok {
					return
				}

				if available := i.monitor.IsAvailable(); !available {
					slog.Warn("Screensaver service is not available, skipping inhibition...")
					i.mu.Lock()
					if i.state != nil {
						i.state.cookie = nil // idle owner changed, so the previous cookie is no longer valid
					}
					i.mu.Unlock()
					continue
				}

				slog.Info("Recovered screensaver service, checking inhibition state...")

				i.mu.Lock()
				if i.state == nil {
					i.mu.Unlock()
					continue
				}

				i.state.cookie = nil // idle owner changed, so the previous cookie is no longer valid

				cookie, err := i.inhibit(i.state.reason)
				if err != nil {
					slog.Error("Failed to re-inhibit idle state", slog.Any("error", err))
				} else {
					i.state.cookie = &cookie
				}

				i.mu.Unlock()
			}
		}
	}()

	return nil
}

func (i *Inhibitor) IsActive() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.state != nil
}

func (i *Inhibitor) Inhibit(reason string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.state != nil {
		err := i.unInhibitUnlocked()
		if err != nil {
			return fmt.Errorf("failed to uninhibit previous inhibition: %w", err)
		}
	}

	i.state = &inhibitionState{
		reason: reason,
		cookie: nil,
	}

	if available := i.monitor.IsAvailable(); !available {
		return nil
	}

	cookie, err := i.inhibit(reason)
	if err != nil {
		return err
	}

	i.state.cookie = &cookie
	return nil
}

func (i *Inhibitor) inhibit(reason string) (uint32, error) {
	call, err := i.client.Object(dbusScreensaverPath).Call(
		dbusInhibitCall,
		daemon.ProgramName,
		reason,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to call Inhibit on screensaver: %w", err)
	}

	var cookie uint32
	if err := call.Store(&cookie); err != nil {
		return 0, fmt.Errorf("failed to store Inhibit cookie: %w", err)
	}

	return cookie, nil
}

func (i *Inhibitor) Uninhibit() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.unInhibitUnlocked()
}

func (i *Inhibitor) unInhibitUnlocked() error {
	if i.state == nil {
		return nil
	}

	data := i.state
	i.state = nil

	if data.cookie == nil {
		return nil
	}

	if _, err := i.client.Object(dbusScreensaverPath).Call(
		dbusUninhibitCall,
		*data.cookie,
	); err != nil {
		return fmt.Errorf("failed to call UnInhibit on screensaver: %w", err)
	}

	return nil
}
