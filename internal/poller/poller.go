// Package poller periodically fetches grill status and hands it to
// subscribers (browsers, alerts, history).
package poller

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/stahnma/gmg/internal/protocol"
)

// Grill is the part of the grill client the poller needs.
type Grill interface {
	Discover(context.Context) error
	Status(context.Context) (protocol.Status, error)
}

// Health describes whether the grill is answering. It is sent to browsers
// as the `connection` event and served at /api/connection.
type Health struct {
	Reachable bool       `json:"reachable"`           // the last poll succeeded
	LastSeen  *time.Time `json:"lastSeen"`            // time of the last good status; null if none yet
	Failures  int        `json:"consecutiveFailures"` // failed polls since the last good one
	Error     string     `json:"error,omitempty"`     // why the last poll failed
}

// Poller fetches status every interval.
type Poller struct {
	grill    Grill
	interval time.Duration
	log      *slog.Logger
	now      func() time.Time

	mu         sync.RWMutex
	subs       []func(protocol.Status)
	healthSubs []func(Health, protocol.Status, bool)
	last       protocol.Status
	hasAny     bool
	health     Health
}

// New returns a poller. log may be nil.
func New(g Grill, interval time.Duration, log *slog.Logger) *Poller {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Poller{grill: g, interval: interval, log: log, now: time.Now}
}

// Subscribe registers fn to be called with every status. Subscribers run
// synchronously in order, so they should be quick.
func (p *Poller) Subscribe(fn func(protocol.Status)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subs = append(p.subs, fn)
}

// OnHealth registers fn to be called after every poll attempt, successful or
// not, with the grill's health and the last good status (ok is false if
// there is none yet). It runs after the status subscribers.
func (p *Poller) OnHealth(fn func(h Health, last protocol.Status, ok bool)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthSubs = append(p.healthSubs, fn)
}

// Health returns the grill's current health.
func (p *Poller) Health() Health {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.health
}

// Last returns the most recent status, if any has been received.
func (p *Poller) Last() (protocol.Status, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.last, p.hasAny
}

// Run polls until ctx is cancelled. Like the original server it waits one
// interval before the first poll, and discovers the grill (when configured
// with a broadcast address) before the first status request.
func (p *Poller) Run(ctx context.Context) {
	discovered := false
	t := time.NewTicker(p.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if !discovered {
			if err := p.grill.Discover(ctx); err != nil {
				p.failed("grill discovery failed", err)
				continue
			}
			discovered = true
		}
		s, err := p.grill.Status(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			p.failed("grill status failed", err)
			continue
		}
		p.succeeded(s)
	}
}

func (p *Poller) succeeded(s protocol.Status) {
	now := p.now()
	p.mu.Lock()
	if !p.health.Reachable && p.hasAny {
		p.log.Info("grill responding again", "after_failures", p.health.Failures)
	}
	p.last, p.hasAny = s, true
	p.health = Health{Reachable: true, LastSeen: &now}
	subs := append([]func(protocol.Status){}, p.subs...)
	p.mu.Unlock()
	for _, fn := range subs {
		fn(s)
	}
	p.notifyHealth()
}

// failed records a failed poll. Only the first failure of an outage is
// logged as a warning, so a grill that is off for hours doesn't flood the log.
func (p *Poller) failed(msg string, err error) {
	p.mu.Lock()
	if p.health.Failures == 0 {
		p.log.Warn(msg, "err", err)
	} else {
		p.log.Debug(msg, "err", err, "failures", p.health.Failures+1)
	}
	p.health.Reachable = false
	p.health.Failures++
	p.health.Error = err.Error()
	p.mu.Unlock()
	p.notifyHealth()
}

func (p *Poller) notifyHealth() {
	p.mu.RLock()
	h, last, ok := p.health, p.last, p.hasAny
	subs := append([]func(Health, protocol.Status, bool){}, p.healthSubs...)
	p.mu.RUnlock()
	for _, fn := range subs {
		fn(h, last, ok)
	}
}
