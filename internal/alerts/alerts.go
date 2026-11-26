// Package alerts turns grill status updates into alerts (target temperature
// reached, low pellets) and delivers them to the browser and Slack.
package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/stahnma/gmg/internal/protocol"
)

// Alert types; also the names of the sound files under /alerts/.
const (
	TypeTargetGrillTempReached = "targetGrillTempReached"
	TypeTargetFoodTempReached  = "targetFoodTempReached"
	TypeLowPelletAlarmActive   = "lowPelletAlarmActive"
	TypeGrillUnreachable       = "grillUnreachable"
	TypeGrillReachable         = "grillReachable"
)

// Alert is sent to the browser as JSON; field names match the original API.
type Alert struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
	Beep   string `json:"beep,omitempty"` // sound file path, e.g. alerts/lowPelletAlarmActive.mp3
	Level  string `json:"level"`          // info, warning or error
}

func beep(alertType string) string { return "alerts/" + alertType + ".mp3" }

// Handler inspects each status and returns an alert when one should fire.
type Handler interface {
	Check(protocol.Status) *Alert
}

// TargetGrillTemp fires once when the grill reaches its target. It re-arms
// when the target changes or the grill is turned off.
type TargetGrillTemp struct{ alerted int }

func (h *TargetGrillTemp) Check(s protocol.Status) *Alert {
	if !s.IsOn {
		h.alerted = 0
		return nil
	}
	if s.CurrentGrillTemp == 0 || s.DesiredGrillTemp == 0 || s.CurrentGrillTemp < s.DesiredGrillTemp ||
		s.DesiredGrillTemp == h.alerted {
		return nil
	}
	h.alerted = s.DesiredGrillTemp
	return &Alert{
		Type:   TypeTargetGrillTempReached,
		Name:   "Target Grill Temperature Reached",
		Reason: fmt.Sprintf("Grill temperature has reached target temperature of %d", s.DesiredGrillTemp),
		Beep:   beep(TypeTargetGrillTempReached),
		Level:  "info",
	}
}

// TargetFoodTemp fires once when the food probe reaches its target. It
// re-arms when the target changes or the grill is turned off.
type TargetFoodTemp struct{ alerted int }

func (h *TargetFoodTemp) Check(s protocol.Status) *Alert {
	if !s.IsOn {
		h.alerted = 0
		return nil
	}
	if s.CurrentFoodTemp == 0 || s.DesiredFoodTemp == 0 || s.CurrentFoodTemp < s.DesiredFoodTemp ||
		s.DesiredFoodTemp == h.alerted {
		return nil
	}
	h.alerted = s.DesiredFoodTemp
	return &Alert{
		Type:   TypeTargetFoodTempReached,
		Name:   "Target Food Temperature Reached",
		Reason: fmt.Sprintf("Food temperature has reached target temperature of %d", s.DesiredFoodTemp),
		Beep:   beep(TypeTargetFoodTempReached),
		Level:  "info",
	}
}

// LowPellet fires when the grill's low-pellet alarm becomes active, and
// again each time it clears and comes back.
type LowPellet struct{ active bool }

func (h *LowPellet) Check(s protocol.Status) *Alert {
	was := h.active
	h.active = s.LowPelletAlarmActive
	if !s.LowPelletAlarmActive || was {
		return nil
	}
	return &Alert{
		Type:   TypeLowPelletAlarmActive,
		Name:   "Grill Pellet Alarm",
		Reason: "The grill is low on pellets. Please inspect the hopper as soon as possible!",
		Beep:   beep(TypeLowPelletAlarmActive),
		Level:  "warning",
	}
}

// Unreachable fires when the grill stops answering for After while it was
// last seen on (or cooling down in fan mode), and fires a recovery alert
// when it answers again. A grill that was off when it went quiet (for
// example unplugged after a cook) doesn't alert. After <= 0 disables it.
type Unreachable struct {
	After   time.Duration
	alerted bool
}

// Check is called after every poll attempt. lastSeen is the time of the last
// good status (nil if none), and last is that status.
func (u *Unreachable) Check(reachable bool, lastSeen *time.Time, last protocol.Status, now time.Time) *Alert {
	if u.After <= 0 {
		return nil
	}
	if reachable {
		if !u.alerted {
			return nil
		}
		u.alerted = false
		return &Alert{
			Type:   TypeGrillReachable,
			Name:   "Grill Responding Again",
			Reason: "The grill is responding again.",
			Level:  "info",
		}
	}
	if u.alerted || lastSeen == nil || !(last.IsOn || last.FanModeActive) || now.Sub(*lastSeen) < u.After {
		return nil
	}
	u.alerted = true
	return &Alert{
		Type: TypeGrillUnreachable,
		Name: "Grill Not Responding",
		Reason: fmt.Sprintf("The grill hasn't responded for %s (last seen at %s, %d °F). Check its power and Wi-Fi.",
			now.Sub(*lastSeen).Round(time.Second), lastSeen.Format("15:04"), last.CurrentGrillTemp),
		Level: "error",
	}
}

// DefaultHandlers returns the standard set of alert handlers.
func DefaultHandlers() []Handler {
	return []Handler{&LowPellet{}, &TargetFoodTemp{}, &TargetGrillTemp{}}
}

// Sender delivers an alert somewhere.
type Sender interface {
	Name() string
	Send(context.Context, Alert) error
}

// Manager runs handlers on each status and delivers resulting alerts.
type Manager struct {
	mu       sync.Mutex
	handlers []Handler
	senders  []Sender
	log      *slog.Logger
	wg       sync.WaitGroup
}

// NewManager returns a manager. log may be nil.
func NewManager(handlers []Handler, senders []Sender, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Manager{handlers: handlers, senders: senders, log: log}
}

// OnStatus evaluates handlers and sends any alerts in the background, so a
// slow sender (e.g. Slack) never delays polling. It returns the alerts.
func (m *Manager) OnStatus(s protocol.Status) []Alert {
	m.mu.Lock()
	var fired []Alert
	for _, h := range m.handlers {
		if a := h.Check(s); a != nil {
			fired = append(fired, *a)
		}
	}
	m.mu.Unlock()
	for _, a := range fired {
		m.Send(a)
	}
	return fired
}

// Send delivers a to every sender in the background.
func (m *Manager) Send(a Alert) {
	m.log.Info("alert", "type", a.Type, "reason", a.Reason)
	for _, snd := range m.senders {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := snd.Send(ctx, a); err != nil {
				m.log.Error("sending alert failed", "sender", snd.Name(), "type", a.Type, "err", err)
			}
		}()
	}
}

// Wait blocks until in-flight sends finish (used in tests and shutdown).
func (m *Manager) Wait() { m.wg.Wait() }

// BrowserSender publishes alerts to connected browsers. With audio disabled
// the sound is removed so the browser shows the alert silently.
type BrowserSender struct {
	Publish      func(event string, payload any)
	AudioEnabled bool
}

func (b BrowserSender) Name() string { return "browser" }

func (b BrowserSender) Send(_ context.Context, a Alert) error {
	if !b.AudioEnabled {
		a.Beep = ""
	}
	b.Publish("alert", a)
	return nil
}

// SlackSender posts alerts to a Slack incoming webhook.
type SlackSender struct {
	WebhookURL string
	Channel    string
	Username   string
	Client     *http.Client // optional
}

func (s SlackSender) Name() string { return "slack" }

var slackEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// Message returns the Slack payload for a.
func (s SlackSender) Message(a Alert) map[string]any {
	return map[string]any{
		"text":       slackEscaper.Replace("*" + a.Name + "*\n" + a.Reason),
		"channel":    s.Channel,
		"link_names": 1,
		"username":   s.Username,
		"icon_emoji": ":meat_on_bone:",
	}
}

func (s SlackSender) Send(ctx context.Context, a Alert) error {
	body, err := json.Marshal(s.Message(a))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("invalid webhook response %d - %s", resp.StatusCode, b)
	}
	return nil
}
