// Package grill is a client for the Green Mountain Grill WiFi UDP protocol.
package grill

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stahnma/gmg/internal/protocol"
)

// BroadcastHost is the default host; the client must discover the grill first.
const BroadcastHost = "255.255.255.255"

// InvalidCommandError is returned for commands the grill can't accept in its
// current state (e.g. setting a temperature while it's off). The HTTP API
// maps it to 400.
type InvalidCommandError struct{ Msg string }

func (e *InvalidCommandError) Error() string { return e.Msg }

func invalid(format string, a ...any) error {
	return &InvalidCommandError{Msg: fmt.Sprintf(format, a...)}
}

// Client talks to one grill. Safe for concurrent use: every command uses its
// own UDP socket, as the original client did.
type Client struct {
	mu        sync.Mutex
	host      string
	broadcast string // configured broadcast address; empty when a grill IP was configured
	port      int
	tries     int
	retry     time.Duration
	log       *slog.Logger
}

// New returns a client. tries is the number of resends after the first send;
// retry is how long to wait for a reply before resending.
func New(host string, port, tries int, retry time.Duration, log *slog.Logger) *Client {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	c := &Client{host: host, port: port, tries: tries, retry: retry, log: log}
	if isBroadcast(host) {
		c.broadcast = host
	}
	return c
}

// isBroadcast reports whether host means "discover the grill", using the
// original client's rule: the address contains "255".
func isBroadcast(host string) bool { return strings.Contains(host, "255") }

// Host returns the grill host (the discovered address after Discover).
func (c *Client) Host() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.host
}

// Status returns the grill's current status.
func (c *Client) Status(ctx context.Context) (protocol.Status, error) {
	b, err := c.send(ctx, protocol.CmdGetStatus, true)
	if err != nil {
		return protocol.Status{}, err
	}
	return protocol.ParseStatus(b)
}

// PowerToggle turns the grill off if it's on, and on if it's off.
func (c *Client) PowerToggle(ctx context.Context) error {
	s, err := c.Status(ctx)
	if err != nil {
		return err
	}
	if s.IsOn {
		return c.powerOff(ctx, s)
	}
	return c.powerOn(ctx, s)
}

// PowerOn turns the grill on. It fails with InvalidCommandError in fan mode.
func (c *Client) PowerOn(ctx context.Context) error {
	s, err := c.Status(ctx)
	if err != nil {
		return err
	}
	return c.powerOn(ctx, s)
}

// PowerOff turns the grill off (no-op if already off).
func (c *Client) PowerOff(ctx context.Context) error {
	s, err := c.Status(ctx)
	if err != nil {
		return err
	}
	return c.powerOff(ctx, s)
}

func (c *Client) powerOn(ctx context.Context, s protocol.Status) error {
	if s.FanModeActive {
		return invalid("Cannot start grill when fan mode is active.")
	}
	return c.command(ctx, protocol.CmdPowerOn, func(n protocol.Status) bool { return n.IsOn })
}

func (c *Client) powerOff(ctx context.Context, s protocol.Status) error {
	if !s.IsOn {
		return nil
	}
	return c.command(ctx, protocol.CmdPowerOff, func(n protocol.Status) bool { return !n.IsOn })
}

// SetGrillTemp sets the grill target (°F). The grill must be on.
func (c *Client) SetGrillTemp(ctx context.Context, f int) error {
	return c.setTemp(ctx, "grill", f, protocol.SetGrillTemp(f), func(n protocol.Status) bool { return n.DesiredGrillTemp == f })
}

// SetFoodTemp sets the food probe target (°F). The grill must be on.
func (c *Client) SetFoodTemp(ctx context.Context, f int) error {
	return c.setTemp(ctx, "food", f, protocol.SetFoodTemp(f), func(n protocol.Status) bool { return n.DesiredFoodTemp == f })
}

func (c *Client) setTemp(ctx context.Context, what string, f int, cmd string, ok func(protocol.Status) bool) error {
	if f < 0 || f > 999 {
		return invalid("Invalid %s temperature: %d", what, f)
	}
	s, err := c.Status(ctx)
	if err != nil {
		return err
	}
	if !s.IsOn {
		return invalid("Cannot set %s temperature when the grill is off!", what)
	}
	return c.command(ctx, cmd, ok)
}

// command sends cmd and accepts either an "OK" reply or a status reply that
// satisfies ok.
func (c *Client) command(ctx context.Context, cmd string, ok func(protocol.Status) bool) error {
	b, err := c.send(ctx, cmd, false)
	if err != nil {
		return err
	}
	if string(b) == protocol.ResponseOK {
		return nil
	}
	if s, err := protocol.ParseStatus(b); err == nil && ok(s) {
		return nil
	}
	return fmt.Errorf("Grill responded with invalid status -> %s", b)
}

// send transmits cmd and waits for a reply from the grill's address,
// resending every retry interval, up to tries+1 sends in total.
func (c *Client) send(ctx context.Context, cmd string, ignoreEmpty bool) ([]byte, error) {
	host := c.Host()
	if host == BroadcastHost {
		return nil, errors.New("Grill host is broadcast address! Please discover first")
	}
	addr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(host, strconv.Itoa(c.port)))
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	attempts := c.tries + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		if _, err := conn.WriteTo([]byte(cmd), addr); err != nil {
			c.log.Warn("grill command send failed", "cmd", cmd, "attempt", attempt, "err", err)
		} else {
			c.log.Debug("grill command sent", "cmd", cmd, "addr", addr, "attempt", attempt)
		}
		deadline := time.Now().Add(c.retry)
		if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
			deadline = d
		}
		for {
			conn.SetReadDeadline(deadline)
			n, from, err := conn.ReadFrom(buf)
			if err != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					break // resend
				}
				return nil, err
			}
			msg := buf[:n]
			if u, ok := from.(*net.UDPAddr); !ok || !u.IP.Equal(addr.IP) {
				continue // not from the grill
			}
			if ignoreEmpty && len(bytes.TrimSpace(msg)) == 0 {
				continue
			}
			c.log.Debug("grill response", "cmd", cmd, "bytes", n)
			return bytes.Clone(msg), nil
		}
	}
	return nil, fmt.Errorf("No response from Grill after [%d] command sent attempts!", attempts)
}

// Discover broadcasts a grill-ID request and switches the client to the
// address of the first grill that answers. It does nothing when a grill IP
// was configured. It can be called again later, e.g. when the grill stops
// answering because it was given a new address.
func (c *Client) Discover(ctx context.Context) error {
	host := c.broadcast
	if host == "" {
		c.log.Debug("grill host is not a broadcast address, skipping discovery", "host", c.Host())
		return nil
	}
	addr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(host, strconv.Itoa(c.port)))
	if err != nil {
		return err
	}
	conn, err := net.ListenPacket("udp4", ":0") // Go enables SO_BROADCAST on UDP sockets
	if err != nil {
		return err
	}
	defer conn.Close()

	req := []byte(protocol.CmdGetGrillID)
	buf := make([]byte, 1024)
	attempts := max(c.tries, 1)
	for attempt := 1; attempt <= attempts; attempt++ {
		if _, err := conn.WriteTo(req, addr); err != nil {
			c.log.Warn("discovery broadcast failed", "addr", addr, "err", err)
		}
		deadline := time.Now().Add(c.retry)
		for {
			conn.SetReadDeadline(deadline)
			n, from, err := conn.ReadFrom(buf)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					break
				}
				return err
			}
			msg := buf[:n]
			if len(bytes.TrimSpace(msg)) == 0 || bytes.Equal(msg, req) {
				continue // empty, or our own broadcast
			}
			u, ok := from.(*net.UDPAddr)
			if !ok {
				continue
			}
			c.mu.Lock()
			old := c.host
			c.host = u.IP.String()
			c.mu.Unlock()
			switch {
			case old == host:
				c.log.Info("grill discovered", "host", u.IP.String(), "id", string(msg))
			case old != u.IP.String():
				c.log.Info("grill address changed", "from", old, "to", u.IP.String(), "id", string(msg))
			default:
				c.log.Debug("grill rediscovered at the same address", "host", old)
			}
			return nil
		}
	}
	return fmt.Errorf("No response from Grill (%s:%d) after [%d] discovery attempts!", host, c.port, attempts)
}
