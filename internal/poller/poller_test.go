package poller

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stahnma/gmg/internal/protocol"
)

type fakeGrill struct {
	discoverErrs atomic.Int32 // fail this many discoveries first
	discovers    atomic.Int32
	statusCalls  atomic.Int32
	failStatus   atomic.Bool
}

func (f *fakeGrill) Discover(context.Context) error {
	f.discovers.Add(1)
	if f.discoverErrs.Load() > 0 {
		f.discoverErrs.Add(-1)
		return errors.New("no grill")
	}
	return nil
}

func (f *fakeGrill) Status(context.Context) (protocol.Status, error) {
	n := f.statusCalls.Add(1)
	if f.failStatus.Load() {
		return protocol.Status{}, errors.New("timeout")
	}
	return protocol.Status{IsOn: true, CurrentGrillTemp: int(n)}, nil
}

func run(t *testing.T, g *fakeGrill, until func() bool) *Poller {
	t.Helper()
	p := New(g, 5*time.Millisecond, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { p.Run(ctx); close(done) }()
	deadline := time.After(2 * time.Second)
	for !until() {
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatal("timed out")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	<-done
	return p
}

func TestFansOutToSubscribers(t *testing.T) {
	g := &fakeGrill{}
	var mu sync.Mutex
	var a, b []int
	p := New(g, 5*time.Millisecond, nil)
	p.Subscribe(func(s protocol.Status) { mu.Lock(); a = append(a, s.CurrentGrillTemp); mu.Unlock() })
	p.Subscribe(func(s protocol.Status) { mu.Lock(); b = append(b, s.CurrentGrillTemp); mu.Unlock() })
	ctx, cancel := context.WithCancel(context.Background())
	go p.Run(ctx)
	for {
		mu.Lock()
		n := len(b)
		mu.Unlock()
		if n >= 3 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	mu.Lock()
	defer mu.Unlock()
	if a[0] != 1 || a[1] != 2 || b[0] != 1 {
		t.Fatalf("subscribers got %v and %v", a, b)
	}
	if s, ok := p.Last(); !ok || s.CurrentGrillTemp < 3 {
		t.Fatalf("Last() = %+v, %v", s, ok)
	}
}

func TestDiscoversOnceRetryingUntilFound(t *testing.T) {
	g := &fakeGrill{}
	g.discoverErrs.Store(2)
	run(t, g, func() bool { return g.statusCalls.Load() >= 3 })
	if n := g.discovers.Load(); n != 3 {
		t.Fatalf("want 2 failed + 1 successful discovery, got %d", n)
	}
}

func TestStatusErrorsDoNotStopPolling(t *testing.T) {
	g := &fakeGrill{}
	g.failStatus.Store(true)
	var got atomic.Int32
	p := run(t, g, func() bool { return g.statusCalls.Load() >= 3 })
	p.Subscribe(func(protocol.Status) { got.Add(1) })
	if _, ok := p.Last(); ok || got.Load() != 0 {
		t.Fatal("failed polls must not publish a status")
	}
}

func TestHealthTracksFailuresAndRecovery(t *testing.T) {
	g := &fakeGrill{}
	p := New(g, 5*time.Millisecond, nil)
	var mu sync.Mutex
	var seen []Health
	p.OnHealth(func(h Health, _ protocol.Status, _ bool) { mu.Lock(); seen = append(seen, h); mu.Unlock() })
	count := func() int { mu.Lock(); defer mu.Unlock(); return len(seen) }

	if h := p.Health(); h.Reachable || h.LastSeen != nil {
		t.Fatalf("before any poll: %+v", h)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { p.Run(ctx); close(done) }()
	defer func() { cancel(); <-done }()
	waitFor := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for !cond() {
			if time.Now().After(deadline) {
				t.Fatal("timed out")
			}
			time.Sleep(time.Millisecond)
		}
	}

	waitFor(func() bool { return p.Health().Reachable })
	seenAt := *p.Health().LastSeen
	g.failStatus.Store(true)
	waitFor(func() bool { return p.Health().Failures >= 2 })
	h := p.Health()
	if h.Reachable || h.Error != "timeout" || h.LastSeen == nil || h.LastSeen.Before(seenAt) {
		t.Fatalf("while failing: %+v", h)
	}
	g.failStatus.Store(false)
	waitFor(func() bool { return p.Health().Reachable })
	if h := p.Health(); h.Failures != 0 || h.Error != "" {
		t.Fatalf("after recovery: %+v", h)
	}
	if count() == 0 {
		t.Fatal("health subscribers not called")
	}
}

func TestFailedDiscoveryIsUnhealthy(t *testing.T) {
	g := &fakeGrill{}
	g.discoverErrs.Store(100)
	p := run(t, g, func() bool { return g.discovers.Load() >= 2 })
	if h := p.Health(); h.Reachable || h.Failures < 2 || h.Error != "no grill" {
		t.Fatalf("got %+v", h)
	}
}
