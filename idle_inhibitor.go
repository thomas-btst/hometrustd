package main

import (
	"fmt"
	"sync"

	godbus "github.com/godbus/dbus/v5"
	"github.com/thomas-btst/hometrustd/dbus"
)

const dbusScreensaverInterface string = "org.freedesktop.ScreenSaver"

const dbusScreensaverPath dbus.Path = "/org/freedesktop/ScreenSaver"

const (
	dbusInhibitCall   dbus.Method = "Inhibit"
	dbusUninhibitCall dbus.Method = "UnInhibit"
)

type IdleInhibitor struct {
	mu     sync.Mutex
	client *dbus.Client
	cookie *uint32
}

func NewIdleInhibitor(conn *godbus.Conn) *IdleInhibitor {
	return &IdleInhibitor{
		client: dbus.NewClient(dbusScreensaverInterface, conn),
	}
}

func (i *IdleInhibitor) IsActive() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.cookie != nil
}

func (i *IdleInhibitor) Inhibit(reason string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.cookie != nil {
		return nil
	}

	object := i.client.Object(dbusScreensaverPath)

	call, err := object.Call(dbusInhibitCall, ProgramName, reason)
	if err != nil {
		return fmt.Errorf("failed to call Inhibit on screensaver: %w", err)
	}

	var cookie uint32
	if err := call.Store(&cookie); err != nil {
		return fmt.Errorf("failed to store Inhibit cookie: %w", err)
	}

	i.cookie = &cookie
	return nil
}

func (i *IdleInhibitor) Uninhibit() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.cookie == nil {
		return nil
	}

	object := i.client.Object(dbusScreensaverPath)

	_, err := object.Call(dbusUninhibitCall, *i.cookie)
	if err != nil {
		return fmt.Errorf("failed to call UnInhibit on screensaver: %w", err)
	}

	i.cookie = nil
	return nil
}
