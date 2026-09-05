package config

import (
	"context"
	"sync"
	"sync/atomic"
)

type Store struct {
	config      atomic.Pointer[Config]
	mu          sync.Mutex
	subscribers map[chan struct{}]struct{}
}

func newStore(config *Config) *Store {
	s := &Store{
		subscribers: make(map[chan struct{}]struct{}),
	}
	s.config.Store(config)
	return s
}

func (s *Store) Update(config *Config) {
	s.config.Store(config)

	s.mu.Lock()
	defer s.mu.Unlock()

	for ch := range s.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (s *Store) Current() *Config {
	return s.config.Load()
}

func (s *Store) Watch(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{}, 1)

	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()

	go func() {
		<-ctx.Done()

		s.mu.Lock()
		delete(s.subscribers, ch)
		s.mu.Unlock()

		close(ch)
	}()

	return ch
}
