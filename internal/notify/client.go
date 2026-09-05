// Package notify provides a client for sending notifications via D-Bus.
package notify

import (
	"fmt"
	"sync"

	godbus "github.com/godbus/dbus/v5"
	"github.com/thomas-btst/hometrustd/internal/config"
	"github.com/thomas-btst/hometrustd/internal/daemon"
	"github.com/thomas-btst/hometrustd/internal/dbus"
)

const (
	dbusInterface string      = "org.freedesktop.Notifications"
	dbusPath      dbus.Path   = "/org/freedesktop/Notifications"
	dbusMethod    dbus.Method = "Notify"
)

const (
	lowUrgency      Urgency = 0
	normalUrgency   Urgency = 1
	criticalUrgency Urgency = 2
)

type Urgency byte

type message struct {
	AppName    string
	Icon       string
	Summary    string
	Body       string
	Urgency    Urgency // 0: low, 1: normal, 2: critical
	Timeout    int32   // Millisecondes, -1 for default
	ReplacesID uint32
}

type Client struct {
	client      *dbus.Client
	configStore *config.Store
	mu          sync.Mutex
	lastID      uint32
}

func NewClient(conn *godbus.Conn, cfgStore *config.Store) *Client {
	return &Client{
		client:      dbus.NewClient(dbusInterface, conn),
		configStore: cfgStore,
	}
}

func (c *Client) send(msg message) (bool, uint32, error) {
	config := c.configStore.Current()
	if config.Quiet {
		return false, 0, nil
	}
	actions := []string{}
	hints := map[string]godbus.Variant{
		"urgency": godbus.MakeVariant(byte(msg.Urgency)),
	}

	call, err := c.client.Object(dbusPath).Call(
		dbusMethod,
		msg.AppName,
		msg.ReplacesID,
		msg.Icon,
		msg.Summary,
		msg.Body,
		actions,
		hints,
		msg.Timeout,
	)
	if err != nil {
		return false, 0, fmt.Errorf("failed to send notification: %w", err)
	}

	var notifID uint32
	if err := call.Store(&notifID); err != nil {
		return false, 0, fmt.Errorf("failed to parse notification ID: %w", err)
	}

	return true, notifID, nil
}

func (c *Client) Send(summary, body string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ok, id, err := c.send(message{
		AppName:    daemon.AppName,
		Icon:       daemon.AppIcon,
		Summary:    summary,
		Body:       body,
		Urgency:    normalUrgency,
		Timeout:    -1,
		ReplacesID: c.lastID,
	})
	if ok {
		c.lastID = id
	}
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}
