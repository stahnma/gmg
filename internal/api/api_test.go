package api

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stahnma/gmg/internal/grill"
	"github.com/stahnma/gmg/internal/poller"
	"github.com/stahnma/gmg/internal/protocol"
	"github.com/stahnma/gmg/internal/store"
)

type fakeGrill struct {
	status   protocol.Status
	err      error
	calls    []string
	lastTemp int
}

func (f *fakeGrill) Status(context.Context) (protocol.Status, error) { return f.status, f.err }
func (f *fakeGrill) PowerToggle(context.Context) error {
	f.calls = append(f.calls, "toggle")
	return f.err
}
func (f *fakeGrill) PowerOn(context.Context) error  { f.calls = append(f.calls, "on"); return f.err }
func (f *fakeGrill) PowerOff(context.Context) error { f.calls = append(f.calls, "off"); return f.err }
func (f *fakeGrill) SetGrillTemp(_ context.Context, t int) error {
	f.calls, f.lastTemp = append(f.calls, "grill"), t
	return f.err
}
func (f *fakeGrill) SetFoodTemp(_ context.Context, t int) error {
	f.calls, f.lastTemp = append(f.calls, "food"), t
	return f.err
}

type fakeHistory struct{ since int64 }

func (f *fakeHistory) History(_ context.Context, since int64) ([]store.Row, error) {
	f.since = since
	food := int64(150)
	return []store.Row{{ID: 1, Timestamp: 1700000000, Grill: 225, Food: &food}}, nil
}

func setup() (*fakeGrill, *fakeHistory, *Server, http.Handler) {
	g := &fakeGrill{status: protocol.Status{State: "on", IsOn: true, CurrentGrillTemp: 225}}
	h := &fakeHistory{}
	s := &Server{Grill: g, History: h, Hub: NewHub(), Static: fstest.MapFS{
		"index.html":                        {Data: []byte("<html>gmg</html>")},
		"assets/index-abc.js":               {Data: []byte("console.log(1)")},
		"alerts/targetGrillTempReached.mp3": {Data: []byte("ID3")},
	}}
	return g, h, s, s.Handler()
}

func do(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestStatus(t *testing.T) {
	_, _, _, h := setup()
	rec := do(h, "GET", "/api/status")
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("%d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	for _, want := range []string{`"state":"on"`, `"isOn":true`, `"currentGrillTemp":225`, `"desiredFoodTemp":0`, `"lowPelletAlarmActive":false`} {
		if !strings.Contains(body, want) {
			t.Errorf("status body missing %s: %s", want, body)
		}
	}
}

func TestCommands(t *testing.T) {
	tests := []struct{ path, call string }{
		{"/api/powertoggle", "toggle"}, {"/api/poweron", "on"}, {"/api/poweroff", "off"},
		{"/api/temperature/grill/225", "grill"}, {"/api/temperature/food/165", "food"},
	}
	for _, tt := range tests {
		g, _, _, h := setup()
		rec := do(h, "PUT", tt.path)
		if rec.Code != 200 || rec.Body.String() != "OK" || len(g.calls) != 1 || g.calls[0] != tt.call {
			t.Errorf("%s: %d %q calls=%v", tt.path, rec.Code, rec.Body.String(), g.calls)
		}
	}
	g, _, _, h := setup()
	do(h, "PUT", "/api/temperature/grill/225")
	if g.lastTemp != 225 {
		t.Errorf("route param not passed: %d", g.lastTemp)
	}
}

func TestErrors(t *testing.T) {
	g, _, _, h := setup()
	g.err = &grill.InvalidCommandError{Msg: "Cannot set grill temperature when the grill is off!"}
	if rec := do(h, "PUT", "/api/temperature/grill/225"); rec.Code != 400 || rec.Body.String() != g.err.Error() {
		t.Errorf("invalid command: %d %q", rec.Code, rec.Body.String())
	}
	g.err = errors.New("No response from Grill after [6] command sent attempts!")
	if rec := do(h, "GET", "/api/status"); rec.Code != 500 || rec.Body.String() != g.err.Error() {
		t.Errorf("grill error: %d %q", rec.Code, rec.Body.String())
	}
	g.err, g.calls = nil, nil
	if rec := do(h, "PUT", "/api/temperature/grill/abc"); rec.Code != 400 || rec.Body.String() != "Invalid temperature: abc" {
		t.Errorf("non-numeric temp: %d %q", rec.Code, rec.Body.String())
	}
	if len(g.calls) != 0 {
		t.Errorf("grill must not be called for an invalid temperature: %v", g.calls)
	}
}

func TestNotFound(t *testing.T) {
	_, _, _, h := setup()
	for _, c := range []struct{ method, path string }{
		{"GET", "/api/nope"}, {"GET", "/api/poweron"}, {"POST", "/api/status"}, {"GET", "/nope.js"},
		{"GET", "/%2e%2e/go.mod"}, {"PUT", "/index.html"}, {"POST", "/api"},
	} {
		if rec := do(h, c.method, c.path); rec.Code != 404 || rec.Body.String() != "Not Found" {
			t.Errorf("%s %s: %d %q", c.method, c.path, rec.Code, rec.Body.String())
		}
	}
}

func TestAPIDocs(t *testing.T) {
	_, _, _, h := setup()
	for _, path := range []string{"/api", "/api/"} {
		rec := do(h, "GET", path)
		if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("GET %s: %d %q", path, rec.Code, rec.Header().Get("Content-Type"))
		}
		// Every endpoint is documented.
		for _, ep := range []string{"/api/status", "/api/powertoggle", "/api/poweron", "/api/poweroff",
			"/api/temperature/grill/", "/api/temperature/food/", "/api/temperature/history", "/api/events"} {
			if !strings.Contains(rec.Body.String(), ep) {
				t.Errorf("GET %s: docs do not mention %s", path, ep)
			}
		}
	}
}

func TestHistory(t *testing.T) {
	_, hist, _, h := setup()
	rec := do(h, "GET", "/api/temperature/history?since=1700000000")
	want := `[{"temperature_log_id":1,"timestamp":1700000000,"grill_temperature":225,"food_temperature":150}]`
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != want || hist.since != 1700000000 {
		t.Fatalf("%d %s since=%d", rec.Code, rec.Body.String(), hist.since)
	}
	if rec := do(h, "GET", "/api/temperature/history"); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("missing since should give []: %s", rec.Body.String())
	}
}

func TestStaticFiles(t *testing.T) {
	_, _, _, h := setup()
	rec := do(h, "GET", "/")
	if rec.Code != 200 || rec.Body.String() != "<html>gmg</html>" {
		t.Errorf("index: %d %q", rec.Code, rec.Body.String())
	}
	rec = do(h, "GET", "/alerts/targetGrillTempReached.mp3")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "audio/mpeg" {
		t.Errorf("mp3: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	rec = do(h, "GET", "/assets/index-abc.js")
	if rec.Code != 200 || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("asset: %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
}

func TestCORS(t *testing.T) {
	g, _, s, _ := setup()
	req := func(h http.Handler, method, path string, hdr ...string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r.Host = "grill.local:3001"
		for i := 0; i < len(hdr); i += 2 {
			r.Header.Set(hdr[i], hdr[i+1])
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}
	h := s.Handler()

	// No Origin (curl, scripts) and same-origin requests work as before.
	if rec := req(h, "PUT", "/api/poweron"); rec.Code != 200 {
		t.Errorf("no origin: %d", rec.Code)
	}
	if rec := req(h, "PUT", "/api/poweron", "Origin", "http://grill.local:3001"); rec.Code != 200 || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("same origin: %d %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}

	// Another site can't send commands, and gets no CORS headers.
	g.calls = nil
	evil := "https://evil.example"
	if rec := req(h, "PUT", "/api/poweron", "Origin", evil); rec.Code != 403 || len(g.calls) != 0 {
		t.Errorf("cross-site PUT: %d, calls %v", rec.Code, g.calls)
	}
	if rec := req(h, "OPTIONS", "/api/poweron", "Origin", evil, "Access-Control-Request-Method", "PUT"); rec.Code != 403 || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("cross-site preflight: %d %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec := req(h, "GET", "/api/status", "Origin", evil); rec.Code != 200 || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("cross-site GET should get no CORS headers: %d %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec := req(h, "PUT", "/api/poweron", "Sec-Fetch-Site", "cross-site"); rec.Code != 403 {
		t.Errorf("cross-site without Origin: %d", rec.Code)
	}

	// Allowed origins get CORS headers.
	s.AllowedOrigins = []string{"https://ui.example"}
	h = s.Handler()
	if rec := req(h, "OPTIONS", "/api/poweron", "Origin", "https://ui.example", "Access-Control-Request-Method", "PUT"); rec.Code != 204 || rec.Header().Get("Access-Control-Allow-Origin") != "https://ui.example" {
		t.Errorf("allowed preflight: %d %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec := req(h, "PUT", "/api/poweron", "Origin", "https://ui.example"); rec.Code != 200 || rec.Header().Get("Vary") != "Origin" {
		t.Errorf("allowed PUT: %d", rec.Code)
	}
	if rec := req(h, "PUT", "/api/poweron", "Origin", evil); rec.Code != 403 {
		t.Errorf("other origin with allow-list: %d", rec.Code)
	}

	// "*" restores the old allow-everything behaviour.
	s.AllowedOrigins = []string{"*"}
	h = s.Handler()
	if rec := req(h, "PUT", "/api/poweron", "Origin", evil); rec.Code != 200 || rec.Header().Get("Access-Control-Allow-Origin") != evil {
		t.Errorf("wildcard: %d %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestConnection(t *testing.T) {
	_, _, s, h := setup()
	if rec := do(h, "GET", "/api/connection"); rec.Code != 404 {
		t.Errorf("without health: %d", rec.Code)
	}
	seen := time.Unix(1764172800, 0).UTC()
	s.Health = func() poller.Health { return poller.Health{Reachable: true, LastSeen: &seen} }
	rec := do(s.Handler(), "GET", "/api/connection")
	if rec.Code != 200 || rec.Body.String() != `{"reachable":true,"lastSeen":"2025-11-26T16:00:00Z","consecutiveFailures":0}` {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}

func TestEvents(t *testing.T) {
	_, _, s, h := setup()
	s.LastStatus = func() (protocol.Status, bool) { return protocol.Status{State: "on", CurrentGrillTemp: 111}, true }
	s.Health = func() poller.Health { return poller.Health{Failures: 3, Error: "timeout"} }
	h = s.Handler()
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	r := bufio.NewReader(resp.Body)
	next := func() (event, data string) {
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			line = strings.TrimRight(line, "\n")
			switch {
			case strings.HasPrefix(line, "event: "):
				event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				data = strings.TrimPrefix(line, "data: ")
			case line == "" && event != "":
				return event, data
			}
		}
	}

	if ev, data := next(); ev != "status" || !strings.Contains(data, `"currentGrillTemp":111`) {
		t.Fatalf("initial event %s %s", ev, data)
	}
	if ev, data := next(); ev != "connection" || !strings.Contains(data, `"reachable":false`) || !strings.Contains(data, `"consecutiveFailures":3`) {
		t.Fatalf("initial connection event %s %s", ev, data)
	}
	for s.Hub.Clients() == 0 {
		time.Sleep(time.Millisecond)
	}
	s.Hub.Publish("alert", map[string]string{"type": "lowPelletAlarmActive"})
	if ev, data := next(); ev != "alert" || data != `{"type":"lowPelletAlarmActive"}` {
		t.Fatalf("alert event %s %s", ev, data)
	}
}

func TestEventsClientDisconnectUnsubscribes(t *testing.T) {
	_, _, s, h := setup()
	srv := httptest.NewServer(h)
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	for s.Hub.Clients() == 0 {
		time.Sleep(time.Millisecond)
	}
	cancel()
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	deadline := time.Now().Add(2 * time.Second)
	for s.Hub.Clients() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("client was not unsubscribed after disconnect")
		}
		time.Sleep(time.Millisecond)
	}
}
