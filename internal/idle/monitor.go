package idle

import (
	"context"
	"fmt"
	"sync"

	godbus "github.com/godbus/dbus/v5"
	"github.com/thomas-btst/hometrustd/internal/dbus"
)

const dbusInterface string = "org.freedesktop.DBus"

const dbusPath dbus.Path = "/org/freedesktop/DBus"

const dbusNameHasOwner dbus.Method = "NameHasOwner"

const dbusOwnerChangedSignal = "NameOwnerChanged"

type Monitor struct {
	client    *dbus.Client
	mu        sync.Mutex
	available bool
}

func NewMonitor(conn *godbus.Conn) *Monitor {
	return &Monitor{
		client: dbus.NewClient(dbusInterface, conn),
	}
}

func (m *Monitor) IsAvailable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.available
}

func (m *Monitor) setAvailable(available bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.available = available
}

func (m *Monitor) Watch(ctx context.Context) (<-chan struct{}, error) {
	matchOptions := []godbus.MatchOption{
		godbus.WithMatchSender(dbusInterface),
		godbus.WithMatchInterface(dbusInterface),
		godbus.WithMatchMember(dbusOwnerChangedSignal),
	}

	ownSignal, err := m.client.Signals(ctx, matchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dbus signals: %w", err)
	}

	initialAvailable, err := m.checkAvailability()
	if err != nil {
		return nil, fmt.Errorf("failed to check screensaver availability: %w", err)
	}

	m.setAvailable(initialAvailable)

	events := make(chan struct{}, 1)

	go func() {
		defer close(events)
		for {
			select {
			case <-ctx.Done():
				return
			case signal, ok := <-ownSignal:
				if !ok {
					return
				}

				if len(signal.Body) == 3 {
					name, okName := signal.Body[0].(string)
					newOwner, okOwner := signal.Body[2].(string)
					if okName && okOwner && name == dbusScreensaverInterface {
						available := newOwner != ""
						m.setAvailable(available)
						select {
						case events <- struct{}{}:
						default:
						}
					}
				}
			}
		}
	}()

	return events, nil
}

func (m *Monitor) checkAvailability() (bool, error) {
	busObj := m.client.Object(dbusPath)

	var hasOwner bool
	busCall, err := busObj.Call(dbusNameHasOwner, dbusScreensaverInterface)
	if err != nil {
		return false, fmt.Errorf("failed to call NameHasOwner: %w", err)
	}

	if err := busCall.Store(&hasOwner); err != nil {
		return false, fmt.Errorf("failed to store NameHasOwner result: %w", err)
	}

	return hasOwner, nil
}
