package dbus

import "github.com/godbus/dbus/v5"

const EmptyPath Path = "/"

type Path dbus.ObjectPath

func (p Path) IsEmpty() bool {
	return p == "" || p == EmptyPath
}
