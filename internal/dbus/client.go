// Package dbus provides a wrapper around the godbus/dbus package to simplify interaction with D-Bus interfaces.
package dbus

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"
)

type Client struct {
	Destination string
	Conn        *dbus.Conn
}

func NewClient(dest string, conn *dbus.Conn) *Client {
	return &Client{
		Destination: dest,
		Conn:        conn,
	}
}

func (r *Client) Object(path Path) *Object {
	object := r.Conn.Object(r.Destination, dbus.ObjectPath(path))
	return newObject(object)
}

func (r *Client) Signals(ctx context.Context, matchOptions []dbus.MatchOption) (chan *dbus.Signal, error) {
	if err := r.Conn.AddMatchSignal(matchOptions...); err != nil {
		return nil, fmt.Errorf("failed to add dbus match signal: %w", err)
	}

	signals := make(chan *dbus.Signal, 10)
	r.Conn.Signal(signals)

	go func() {
		<-ctx.Done()
		_ = r.Conn.RemoveMatchSignal(matchOptions...)

		r.Conn.RemoveSignal(signals)
	}()

	return signals, nil
}
