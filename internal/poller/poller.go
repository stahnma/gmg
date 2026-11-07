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

// Poller fetches status every interval.
type Poller struct {
	grill    Grill
	interval time.Duration
	log      *slog.Logger

	mu     sync.RWMutex
	subs   []func(protocol.Status)
	last   protocol.Status
	hasAny bool
}

// New returns a poller. log may be nil.
func New(g Grill, interval time.Duration, log *slog.Logger) *Poller {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Poller{grill: g, interval: interval, log: log}
}

// Subscribe registers fn to be called with every status. Subscribers run
// synchronously in order, so they should be quick.
func (p *Poller) Subscribe(fn func(protocol.Status)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subs = append(p.subs, fn)
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
				p.log.Warn("grill discovery failed", "err", err)
				continue
			}
			discovered = true
		}
		s, err := p.grill.Status(ctx)
		if err != nil {
			if ctx.Err() == nil {
				p.log.Warn("grill status failed", "err", err)
			}
			continue
		}
		p.mu.Lock()
		p.last, p.hasAny = s, true
		subs := append([]func(protocol.Status){}, p.subs...)
		p.mu.Unlock()
		for _, fn := range subs {
			fn(s)
		}
	}
}
