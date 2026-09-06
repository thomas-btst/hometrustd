package dbus

import (
	"context"
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
)

const dbusInterface string = "org.freedesktop.DBus"

const dbusPath Path = "/org/freedesktop/DBus"

const dbusNameHasOwner Method = "NameHasOwner"

const dbusOwnerChangedSignal = "NameOwnerChanged"

type targetState struct {
	available   bool
	subscribers map[chan struct{}]struct{}
	version     uint64
}

type Monitor struct {
	client       *Client
	mu           sync.RWMutex
	targetsState map[string]*targetState
}

func NewMonitor(conn *dbus.Conn) *Monitor {
	return &Monitor{
		client:       NewClient(dbusInterface, conn),
		targetsState: make(map[string]*targetState),
	}
}

func (m *Monitor) IsAvailable(targetInterface string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if target, ok := m.targetsState[targetInterface]; ok {
		return target.available
	}
	return false
}

func (m *Monitor) Start(ctx context.Context) error {
	matchOptions := []dbus.MatchOption{
		dbus.WithMatchSender(dbusInterface),
		dbus.WithMatchInterface(dbusInterface),
		dbus.WithMatchMember(dbusOwnerChangedSignal),
	}

	ownSignal, err := m.client.Signals(ctx, matchOptions)
	if err != nil {
		return fmt.Errorf("failed to initialize dbus signals: %w", err)
	}

	go func() {
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
					if okName && okOwner {
						m.dispatch(name, newOwner != "")
					}
				}
			}
		}
	}()

	return nil
}

func (m *Monitor) dispatch(targetInterface string, available bool) {
	m.mu.RLock() // Read-First guard
	_, ok := m.targetsState[targetInterface]
	if !ok {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	targetState, ok := m.targetsState[targetInterface]
	if !ok {
		return
	}

	targetState.available = available
	targetState.version++

	for sub := range targetState.subscribers {
		select {
		case sub <- struct{}{}:
		default:
		}
	}
}

func (m *Monitor) deleteSubscriber(targetInterface string, sub chan struct{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	target, ok := m.targetsState[targetInterface]
	if !ok {
		return
	}
	delete(target.subscribers, sub)
	if len(target.subscribers) == 0 {
		delete(m.targetsState, targetInterface)
	}
}

func (m *Monitor) Watch(ctx context.Context, targetInterface string) (<-chan struct{}, error) {
	events := make(chan struct{}, 1)

	m.mu.Lock()
	target, ok := m.targetsState[targetInterface]
	if !ok {
		target = &targetState{
			subscribers: make(map[chan struct{}]struct{}),
		}
		m.targetsState[targetInterface] = target
	}
	target.subscribers[events] = struct{}{}
	startVersion := target.version
	m.mu.Unlock()

	initialAvailable, err := m.checkAvailability(targetInterface)
	if err != nil {
		m.deleteSubscriber(targetInterface, events)
		return nil, fmt.Errorf("failed to check availability for interface %s: %w", targetInterface, err)
	}

	m.mu.Lock()
	if target.version == startVersion {
		target.available = initialAvailable
		target.version++
	}
	m.mu.Unlock()

	go func() {
		<-ctx.Done()
		m.deleteSubscriber(targetInterface, events)
		close(events)
	}()

	return events, nil
}

func (m *Monitor) checkAvailability(targetInterface string) (bool, error) {
	busObj := m.client.Object(dbusPath)

	var hasOwner bool
	busCall, err := busObj.Call(dbusNameHasOwner, targetInterface)
	if err != nil {
		return false, fmt.Errorf("failed to call NameHasOwner: %w", err)
	}

	if err := busCall.Store(&hasOwner); err != nil {
		return false, fmt.Errorf("failed to store NameHasOwner result: %w", err)
	}

	return hasOwner, nil
}
