package alerts

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stahnma/gmg/internal/protocol"
)

func grill(cur, want int) protocol.Status {
	return protocol.Status{IsOn: true, CurrentGrillTemp: cur, DesiredGrillTemp: want}
}

func food(cur, want int) protocol.Status {
	return protocol.Status{IsOn: true, CurrentFoodTemp: cur, DesiredFoodTemp: want}
}

// fires runs statuses through h and reports which ones produced an alert.
func fires(h Handler, statuses ...protocol.Status) []bool {
	var out []bool
	for _, s := range statuses {
		out = append(out, h.Check(s) != nil)
	}
	return out
}

func eq(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTargetGrillTemp(t *testing.T) {
	tests := []struct {
		name     string
		statuses []protocol.Status
		want     []bool
	}{
		{"fires once at target", []protocol.Status{grill(200, 225), grill(225, 225), grill(230, 225)}, []bool{false, true, false}},
		{"not when off", []protocol.Status{{IsOn: false, CurrentGrillTemp: 300, DesiredGrillTemp: 225}}, []bool{false}},
		{"not without a target", []protocol.Status{grill(300, 0)}, []bool{false}},
		{"re-arms on new target", []protocol.Status{grill(225, 225), grill(250, 250), grill(250, 225)}, []bool{true, true, true}},
		{"dip below target does not re-fire", []protocol.Status{grill(225, 225), grill(220, 225), grill(226, 225)}, []bool{true, false, false}},
		// Fix: the Node version never alerted again for the same target, even after a power cycle.
		{"re-arms after power off", []protocol.Status{grill(225, 225), {IsOn: false}, grill(225, 225)}, []bool{true, false, true}},
	}
	for _, tt := range tests {
		if got := fires(&TargetGrillTemp{}, tt.statuses...); !eq(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
	a := (&TargetGrillTemp{}).Check(grill(225, 225))
	want := Alert{Type: "targetGrillTempReached", Name: "Target Grill Temperature Reached",
		Reason: "Grill temperature has reached target temperature of 225", Beep: "alerts/targetGrillTempReached.mp3", Level: "info"}
	if *a != want {
		t.Errorf("alert %+v", *a)
	}
}

func TestTargetFoodTemp(t *testing.T) {
	got := fires(&TargetFoodTemp{}, food(150, 165), food(165, 165), food(170, 165), food(170, 170), protocol.Status{}, food(170, 170))
	if want := []bool{false, true, false, true, false, true}; !eq(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if fires(&TargetFoodTemp{}, food(0, 165))[0] { // no probe connected
		t.Error("must not fire without a probe reading")
	}
	a := (&TargetFoodTemp{}).Check(food(165, 165))
	if a.Reason != "Food temperature has reached target temperature of 165" || a.Beep != "alerts/targetFoodTempReached.mp3" {
		t.Errorf("alert %+v", *a)
	}
}

func TestLowPellet(t *testing.T) {
	low := protocol.Status{IsOn: true, LowPelletAlarmActive: true}
	ok := protocol.Status{IsOn: true}
	// Fix: the Node version alerted only once per server run.
	got := fires(&LowPellet{}, ok, low, low, ok, low)
	if want := []bool{false, true, false, false, true}; !eq(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if a := (&LowPellet{}).Check(low); a.Level != "warning" || a.Name != "Grill Pellet Alarm" {
		t.Errorf("alert %+v", *a)
	}
}

type recordSender struct {
	mu   sync.Mutex
	got  []Alert
	fail bool
}

func (r *recordSender) Name() string { return "record" }
func (r *recordSender) Send(_ context.Context, a Alert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, a)
	if r.fail {
		return errors.New("boom")
	}
	return nil
}

func TestManagerDeliversToAllSenders(t *testing.T) {
	a, b := &recordSender{}, &recordSender{fail: true}
	m := NewManager(DefaultHandlers(), []Sender{a, b}, nil)
	s := protocol.Status{IsOn: true, LowPelletAlarmActive: true, CurrentGrillTemp: 225, DesiredGrillTemp: 225}
	if fired := m.OnStatus(s); len(fired) != 2 {
		t.Fatalf("want low pellet + grill target, got %+v", fired)
	}
	m.Wait()
	if len(a.got) != 2 || len(b.got) != 2 {
		t.Fatalf("each sender should get both alerts (a failing sender doesn't affect others): %d, %d", len(a.got), len(b.got))
	}
	if len(m.OnStatus(s)) != 0 {
		t.Fatal("same status again should not re-fire")
	}
}

func TestBrowserSenderAudio(t *testing.T) {
	a := Alert{Type: "x", Beep: "alerts/x.mp3"}
	for _, audio := range []bool{true, false} {
		var got Alert
		BrowserSender{AudioEnabled: audio, Publish: func(ev string, p any) {
			if ev != "alert" {
				t.Errorf("event %q", ev)
			}
			got = p.(Alert)
		}}.Send(context.Background(), a)
		if (got.Beep != "") != audio {
			t.Errorf("audio=%v: beep %q", audio, got.Beep)
		}
	}
	b, _ := json.Marshal(Alert{Type: "x"})
	if strings.Contains(string(b), "beep") {
		t.Errorf("empty beep must be omitted from JSON: %s", b)
	}
}

func TestSlackSender(t *testing.T) {
	var body map[string]any
	var ctype string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctype = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &body)
	}))
	defer srv.Close()

	s := SlackSender{WebhookURL: srv.URL, Channel: "#alerts", Username: "grill-bot"}
	if err := s.Send(context.Background(), Alert{Name: "Low <pellets>", Reason: "A & B"}); err != nil {
		t.Fatal(err)
	}
	if ctype != "application/json" {
		t.Errorf("content type %q", ctype)
	}
	if body["text"] != "*Low &lt;pellets&gt;*\nA &amp; B" || body["channel"] != "#alerts" ||
		body["username"] != "grill-bot" || body["icon_emoji"] != ":meat_on_bone:" || body["link_names"] != float64(1) {
		t.Errorf("payload %+v", body)
	}
}

func TestSlackSenderErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid_token", http.StatusForbidden)
	}))
	defer srv.Close()
	err := SlackSender{WebhookURL: srv.URL}.Send(context.Background(), Alert{})
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "invalid_token") {
		t.Fatalf("got %v", err)
	}
}

func TestUnreachable(t *testing.T) {
	t0 := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	on := protocol.Status{IsOn: true, CurrentGrillTemp: 225}
	at := func(s int) time.Time { return t0.Add(time.Duration(s) * time.Second) }
	seen := &t0

	u := &Unreachable{After: time.Minute}
	if a := u.Check(false, seen, on, at(30)); a != nil {
		t.Fatal("fired before the window")
	}
	a := u.Check(false, seen, on, at(61))
	if a == nil || a.Type != TypeGrillUnreachable || a.Level != "error" || !strings.Contains(a.Reason, "1m1s") || !strings.Contains(a.Reason, "225") {
		t.Fatalf("got %+v", a)
	}
	if u.Check(false, seen, on, at(120)) != nil {
		t.Fatal("fired twice for one outage")
	}
	now := at(130)
	if a := u.Check(true, &now, on, now); a == nil || a.Type != TypeGrillReachable {
		t.Fatalf("recovery: %+v", a)
	}
	if u.Check(true, &now, on, at(135)) != nil {
		t.Fatal("recovery fired twice")
	}
	if u.Check(false, &now, on, at(200)) == nil {
		t.Fatal("did not re-arm after recovery")
	}

	fan := protocol.Status{FanModeActive: true}
	if (&Unreachable{After: time.Minute}).Check(false, seen, fan, at(61)) == nil {
		t.Error("should fire while cooling down in fan mode")
	}
	if (&Unreachable{After: time.Minute}).Check(false, seen, protocol.Status{}, at(600)) != nil {
		t.Error("fired for a grill that was off")
	}
	if (&Unreachable{After: time.Minute}).Check(false, nil, on, at(600)) != nil {
		t.Error("fired for a grill never seen")
	}
	if (&Unreachable{}).Check(false, seen, on, at(600)) != nil {
		t.Error("fired while disabled")
	}
	if (&Unreachable{After: time.Minute}).Check(true, seen, on, at(5)) != nil {
		t.Error("recovery fired without an outage")
	}
}

func TestTempDrop(t *testing.T) {
	clock := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	h := &TempDrop{Degrees: 50, For: 5 * time.Minute, Now: func() time.Time { return clock }}
	step := func(cur, want int, advance time.Duration) bool {
		clock = clock.Add(advance)
		return h.Check(grill(cur, want)) != nil
	}

	// Warm-up from cold never fires, however long it takes.
	if step(100, 225, 0) || step(110, 225, time.Hour) {
		t.Fatal("fired during warm-up")
	}
	// Reaches the target, then the fire goes out.
	if step(220, 225, time.Minute) || step(170, 225, time.Minute) || step(160, 225, 4*time.Minute) {
		t.Fatal("fired too early")
	}
	if !step(150, 225, time.Minute) {
		t.Fatal("did not fire after 5 minutes 50+ °F below target")
	}
	if step(140, 225, 10*time.Minute) {
		t.Fatal("fired twice for one drop")
	}
	// Recovers, drops again: fires again.
	if step(225, 225, time.Minute) || step(170, 225, time.Minute) {
		t.Fatal("unexpected")
	}
	if !step(170, 225, 5*time.Minute) {
		t.Fatal("did not re-arm after recovery")
	}
	// A brief dip that recovers before the window doesn't fire.
	step(225, 225, time.Minute)
	if step(160, 225, time.Minute) || step(200, 225, 4*time.Minute) || step(160, 225, 2*time.Minute) {
		t.Fatal("dip timer should restart after climbing back above the threshold")
	}

	// Raising the target disarms until the new target is reached.
	h2 := &TempDrop{Degrees: 50, For: time.Minute, Now: func() time.Time { return clock }}
	h2.Check(grill(225, 225))
	clock = clock.Add(time.Hour)
	if h2.Check(grill(225, 350)) != nil || h2.Check(grill(230, 350)) != nil {
		t.Fatal("fired after raising the target")
	}

	// Off, or disabled: never.
	h3 := &TempDrop{Degrees: 0, For: time.Minute}
	h3.Check(grill(225, 225))
	if h3.Check(grill(100, 225)) != nil {
		t.Fatal("fired while disabled")
	}
	h4 := &TempDrop{Degrees: 50, For: 0}
	h4.Check(grill(225, 225))
	if h4.Check(protocol.Status{CurrentGrillTemp: 100, DesiredGrillTemp: 225}) != nil || h4.Check(grill(100, 225)) != nil {
		t.Fatal("power off should disarm")
	}
	h5 := &TempDrop{Degrees: 50}
	h5.Check(grill(225, 225))
	if a := h5.Check(grill(150, 225)); a == nil || a.Level != "error" || !strings.Contains(a.Reason, "75 °F below its target of 225") {
		t.Fatalf("alert %+v", a)
	}
}
