// Package dbus provides a wrapper around the godbus/dbus package to simplify interaction with D-Bus interfaces.
package dbus

import (
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
