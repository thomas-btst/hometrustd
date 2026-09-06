package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/thomas-btst/hometrustd/internal/config"
	"github.com/thomas-btst/hometrustd/internal/network"
)

type event int

const (
	defaultEvent event = iota
	networkEvent event = iota
	configEvent  event = iota
)

type NetworkWatcher interface {
	Watch(ctx context.Context) (<-chan struct{}, error)
	State() network.State
}

type ConfigWatcher interface {
	Watch(ctx context.Context) <-chan struct{}
	Current() *config.Config
}

type TrustState struct {
	Trusted bool
	Name    string
}

func NewUntrustedState() TrustState {
	return TrustState{
		Trusted: false,
	}
}

func NewTrustedState(name string) TrustState {
	return TrustState{
		Trusted: true,
		Name:    name,
	}
}

type Monitor struct {
	networkWatcher NetworkWatcher
	configWatcher  ConfigWatcher
	mu             sync.Mutex
	state          TrustState
}

func NewMonitor(netWatcher NetworkWatcher, cfgWatcher ConfigWatcher) *Monitor {
	return &Monitor{
		networkWatcher: netWatcher,
		configWatcher:  cfgWatcher,
		state:          NewUntrustedState(),
	}
}

func (m *Monitor) State() TrustState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *Monitor) Watch(ctx context.Context) (<-chan struct{}, error) {
	netEvents, err := m.networkWatcher.Watch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start network monitor watch: %w", err)
	}

	cfgEvents := m.configWatcher.Watch(ctx)

	initialState := m.evaluateState(defaultEvent)
	m.mu.Lock()
	m.state = initialState
	m.mu.Unlock()

	events := make(chan struct{}, 1)
	go func() {
		defer close(events)

		for {
			var event event
			var ok bool

			select {
			case <-ctx.Done():
				return
			case _, ok = <-cfgEvents:
				event = configEvent
			case _, ok = <-netEvents:
				event = networkEvent
			}
			if !ok {
				return
			}

			state := m.evaluateState(event)

			m.mu.Lock()
			if state != m.state {
				m.state = state
				select {
				case events <- struct{}{}:
				default:
				}
			}
			m.mu.Unlock()
		}
	}()

	return events, nil
}

func (m *Monitor) evaluateState(event event) TrustState {
	if event == networkEvent || event == defaultEvent {
		state := m.networkWatcher.State()
		if state.Connected {
			slog.Info(
				"Connected to Wi-Fi network",
				slog.String("bssid", state.BSSID.String()),
				slog.String("ssid", state.SSID),
			)
		} else {
			slog.Info("Disconnected from Wi-Fi network")
		}
	}

	netState := m.networkWatcher.State()

	if !netState.Connected {
		return NewUntrustedState()
	}

	trustNets := m.configWatcher.Current().TrustedNetworks
	alias, trusted := trustNets.BSSIDs[netState.BSSID]

	if !trusted {
		return NewUntrustedState()
	}

	name := netState.SSID
	if alias != "" {
		name = alias
	}

	return NewTrustedState(name)
}
