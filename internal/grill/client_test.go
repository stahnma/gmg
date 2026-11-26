package grill

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stahnma/gmg/internal/emulator"
	"github.com/stahnma/gmg/internal/protocol"
)

const fastRetry = 50 * time.Millisecond

// startEmulator runs the Go emulator on a loopback port.
func startEmulator(t *testing.T) (*emulator.Grill, int) {
	t.Helper()
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	g := emulator.New()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { emulator.Serve(ctx, conn, g, slog.New(slog.DiscardHandler)); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	return g, conn.LocalAddr().(*net.UDPAddr).Port
}

// fakePeer answers each request with whatever reply returns (nil = no reply).
// Replies are sent from a socket bound to replyFrom (defaults to the
// listening socket). It counts requests received.
func fakePeer(t *testing.T, listen string, replyFrom string, reply func(req string) []byte) (port int, requests *atomic.Int32) {
	t.Helper()
	conn, err := net.ListenPacket("udp4", listen)
	if err != nil {
		t.Fatal(err)
	}
	out := conn
	if replyFrom != "" {
		if out, err = net.ListenPacket("udp4", replyFrom); err != nil {
			t.Fatal(err)
		}
	}
	requests = &atomic.Int32{}
	go func() {
		buf := make([]byte, 1024)
		for {
			n, from, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			requests.Add(1)
			if r := reply(string(buf[:n])); r != nil {
				out.WriteTo(r, from)
			}
		}
	}()
	t.Cleanup(func() { conn.Close(); out.Close() })
	return conn.LocalAddr().(*net.UDPAddr).Port, requests
}

func newClient(port int) *Client { return New("127.0.0.1", port, 2, fastRetry, nil) }

func TestAgainstEmulator(t *testing.T) {
	g, port := startEmulator(t)
	c := newClient(port)
	ctx := context.Background()

	s, err := c.Status(ctx)
	if err != nil || s.IsOn || s.CurrentGrillTemp != 90 {
		t.Fatalf("initial status %+v, %v", s, err)
	}

	var inv *InvalidCommandError
	if err := c.SetGrillTemp(ctx, 225); !errors.As(err, &inv) || inv.Msg != "Cannot set grill temperature when the grill is off!" {
		t.Fatalf("set grill while off: %v", err)
	}
	if err := c.SetFoodTemp(ctx, 165); !errors.As(err, &inv) || inv.Msg != "Cannot set food temperature when the grill is off!" {
		t.Fatalf("set food while off: %v", err)
	}

	if err := c.PowerToggle(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.SetGrillTemp(ctx, 225); err != nil {
		t.Fatal(err)
	}
	if err := c.SetFoodTemp(ctx, 50); err != nil {
		t.Fatal(err)
	}
	g.Tick()
	s, _ = c.Status(ctx)
	if !s.IsOn || s.DesiredGrillTemp != 225 || s.DesiredFoodTemp != 50 || s.CurrentGrillTemp != 92 {
		t.Fatalf("after commands: %+v", s)
	}

	if err := c.PowerOn(ctx); err != nil { // already on: still OK
		t.Fatal(err)
	}
	if err := c.PowerOff(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.PowerOff(ctx); err != nil { // already off: no-op
		t.Fatal(err)
	}
	if s, _ := c.Status(ctx); s.IsOn {
		t.Fatal("grill should be off")
	}
}

func TestNoResponseRetriesThenFails(t *testing.T) {
	port, reqs := fakePeer(t, "127.0.0.1:0", "", func(string) []byte { return nil })
	_, err := New("127.0.0.1", port, 2, fastRetry, nil).Status(context.Background())
	if err == nil || err.Error() != "No response from Grill after [3] command sent attempts!" {
		t.Fatalf("got %v", err)
	}
	if n := reqs.Load(); n != 3 {
		t.Fatalf("want 3 sends (tries+1), got %d", n)
	}
}

func TestIgnoresRepliesFromOtherAddresses(t *testing.T) {
	ok := protocol.RawStatus{State: protocol.StateOn}.Encode()
	port, _ := fakePeer(t, "127.0.0.1:0", "127.0.0.2:0", func(string) []byte { return ok })
	if _, err := New("127.0.0.1", port, 0, fastRetry, nil).Status(context.Background()); err == nil {
		t.Fatal("a reply from a different address must not be accepted")
	}
}

func TestIgnoresEmptyStatusReplies(t *testing.T) {
	var n atomic.Int32
	status := protocol.RawStatus{GrillTemp: 200, State: protocol.StateOn}.Encode()
	port, _ := fakePeer(t, "127.0.0.1:0", "", func(string) []byte {
		if n.Add(1) == 1 {
			return []byte("  ")
		}
		return status
	})
	s, err := New("127.0.0.1", port, 2, fastRetry, nil).Status(context.Background())
	if err != nil || s.CurrentGrillTemp != 200 {
		t.Fatalf("got %+v, %v", s, err)
	}
}

func TestValidatesCommandResult(t *testing.T) {
	on := protocol.RawStatus{State: protocol.StateOn, GrillSetTemp: 180}.Encode()
	port, _ := fakePeer(t, "127.0.0.1:0", "", func(req string) []byte { return on }) // never reports 225
	err := newClient(port).SetGrillTemp(context.Background(), 225)
	if err == nil || !strings.HasPrefix(err.Error(), "Grill responded with invalid status") {
		t.Fatalf("got %v", err)
	}
	// A status reply that does satisfy the command is accepted instead of OK.
	port, _ = fakePeer(t, "127.0.0.1:0", "", func(req string) []byte { return on })
	if err := newClient(port).SetGrillTemp(context.Background(), 180); err != nil {
		t.Fatalf("matching status should be accepted: %v", err)
	}
}

func TestPowerOnRefusedInFanMode(t *testing.T) {
	fan := protocol.RawStatus{State: protocol.StateFan}.Encode()
	port, reqs := fakePeer(t, "127.0.0.1:0", "", func(string) []byte { return fan })
	var inv *InvalidCommandError
	for _, f := range []func(*Client) error{
		func(c *Client) error { return c.PowerOn(context.Background()) },
		func(c *Client) error { return c.PowerToggle(context.Background()) },
	} {
		if err := f(newClient(port)); !errors.As(err, &inv) || inv.Msg != "Cannot start grill when fan mode is active." {
			t.Fatalf("got %v", err)
		}
	}
	if reqs.Load() != 2 {
		t.Fatalf("only status requests should be sent, got %d", reqs.Load())
	}
}

func TestTemperatureLimits(t *testing.T) {
	var inv *InvalidCommandError
	c := newClient(1) // nothing listens: out-of-range values must fail before any send
	for _, f := range []int{149, 551, 1000, -1} {
		if err := c.SetGrillTemp(context.Background(), f); !errors.As(err, &inv) || !strings.Contains(inv.Msg, "between 150 and 550") {
			t.Errorf("grill %d: %v", f, err)
		}
	}
	if err := c.SetFoodTemp(context.Background(), 301); !errors.As(err, &inv) || !strings.Contains(inv.Msg, "food temperature must be between 0 and 300") {
		t.Errorf("food: %v", err)
	}
	c.SetLimits(Limits{Grill: Range{Min: 180, Max: 500}, Food: Range{Min: 100, Max: 200}})
	if err := c.SetGrillTemp(context.Background(), 170); !errors.As(err, &inv) || !strings.Contains(inv.Msg, "between 180 and 500") {
		t.Errorf("custom limits: %v", err)
	}
	if c.Limits().Food.Max != 200 {
		t.Errorf("Limits() = %+v", c.Limits())
	}
}

func TestBroadcastHostRequiresDiscovery(t *testing.T) {
	_, err := New(BroadcastHost, 8080, 0, fastRetry, nil).Status(context.Background())
	if err == nil || !strings.Contains(err.Error(), "discover first") {
		t.Fatalf("got %v", err)
	}
}

func TestDiscover(t *testing.T) {
	// The whole 127/8 block is local on Linux, so 127.0.0.255 (a "255"
	// address) reaches a socket bound to 0.0.0.0 like a broadcast would.
	port, _ := fakePeer(t, "0.0.0.0:0", "", func(req string) []byte {
		if req == protocol.CmdGetGrillID {
			return []byte(emulator.GrillID)
		}
		return nil
	})
	c := New("127.0.0.255", port, 2, fastRetry, nil)
	if err := c.Discover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if h := c.Host(); h == "127.0.0.255" || !strings.HasPrefix(h, "127.") {
		t.Fatalf("host should switch to the responder's address, got %q", h)
	}
}

func TestDiscoverSkippedForUnicastHost(t *testing.T) {
	c := New("192.168.1.50", 8080, 2, fastRetry, nil)
	if err := c.Discover(context.Background()); err != nil || c.Host() != "192.168.1.50" {
		t.Fatalf("got %v, host %q", err, c.Host())
	}
}

func TestDiscoverNoResponse(t *testing.T) {
	port, _ := fakePeer(t, "0.0.0.0:0", "", func(string) []byte { return nil })
	err := New("127.0.0.255", port, 2, fastRetry, nil).Discover(context.Background())
	if err == nil || !strings.Contains(err.Error(), "after [2] discovery attempts") {
		t.Fatalf("got %v", err)
	}
}

func TestRediscoverAfterAddressChange(t *testing.T) {
	// One listener; replies come from 127.0.0.2 first, then from 127.0.0.3,
	// as if the grill had been given a new address.
	conn, err := net.ListenPacket("udp4", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	var outs [2]net.PacketConn
	for i, a := range []string{"127.0.0.2:0", "127.0.0.3:0"} {
		if outs[i], err = net.ListenPacket("udp4", a); err != nil {
			t.Skipf("can't bind %s: %v", a, err)
		}
	}
	t.Cleanup(func() { conn.Close(); outs[0].Close(); outs[1].Close() })
	var which atomic.Int32
	go func() {
		buf := make([]byte, 1024)
		for {
			n, from, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			if string(buf[:n]) == protocol.CmdGetGrillID {
				outs[which.Load()].WriteTo([]byte(emulator.GrillID), from)
			}
		}
	}()

	c := New("127.0.0.255", conn.LocalAddr().(*net.UDPAddr).Port, 2, fastRetry, nil)
	if err := c.Discover(context.Background()); err != nil || c.Host() != "127.0.0.2" {
		t.Fatalf("first discovery: %v, host %q", err, c.Host())
	}
	which.Store(1)
	if err := c.Discover(context.Background()); err != nil || c.Host() != "127.0.0.3" {
		t.Fatalf("second discovery should follow the grill: %v, host %q", err, c.Host())
	}
}

func TestFailedRediscoveryKeepsLastAddress(t *testing.T) {
	port, _ := fakePeer(t, "0.0.0.0:0", "127.0.0.2:0", func(req string) []byte {
		if req == protocol.CmdGetGrillID {
			return []byte(emulator.GrillID)
		}
		return nil
	})
	c := New("127.0.0.255", port, 1, fastRetry, nil)
	if err := c.Discover(context.Background()); err != nil {
		t.Fatal(err)
	}
	c.port = 1 // nothing answers there
	if err := c.Discover(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
	if c.Host() != "127.0.0.2" {
		t.Fatalf("host %q", c.Host())
	}
}
