// Package api serves the HTTP API, the live event stream and the web UI.
package api

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/stahnma/gmg/internal/grill"
	"github.com/stahnma/gmg/internal/poller"
	"github.com/stahnma/gmg/internal/protocol"
	"github.com/stahnma/gmg/internal/store"
)

// Grill is the grill client as used by the API.
type Grill interface {
	Status(context.Context) (protocol.Status, error)
	PowerToggle(context.Context) error
	PowerOn(context.Context) error
	PowerOff(context.Context) error
	SetGrillTemp(context.Context, int) error
	SetFoodTemp(context.Context, int) error
}

// History is the temperature history store as used by the API.
type History interface {
	History(ctx context.Context, since int64) ([]store.Row, error)
}

// Server holds the API's dependencies.
type Server struct {
	Grill      Grill
	History    History
	Hub        *Hub
	LastStatus func() (protocol.Status, bool) // sent to browsers on connect; optional
	Health     func() poller.Health           // grill reachability; optional
	Static     fs.FS                          // web UI and sounds; optional
	Log        *slog.Logger
	KeepAlive  time.Duration // SSE keep-alive comment interval; default 25s
	// AllowedOrigins lists other origins (e.g. "https://grill.example.com")
	// allowed to call the API from a browser; "*" allows any. Same-origin
	// requests are always allowed.
	AllowedOrigins []string
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	if s.Log == nil {
		s.Log = slog.New(slog.DiscardHandler)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/status", s.status)
	mux.HandleFunc("PUT /api/powertoggle", s.command(func(ctx context.Context, _ *http.Request) error { return s.Grill.PowerToggle(ctx) }))
	mux.HandleFunc("PUT /api/poweron", s.command(func(ctx context.Context, _ *http.Request) error { return s.Grill.PowerOn(ctx) }))
	mux.HandleFunc("PUT /api/poweroff", s.command(func(ctx context.Context, _ *http.Request) error { return s.Grill.PowerOff(ctx) }))
	mux.HandleFunc("PUT /api/temperature/grill/{tempF}", s.command(func(ctx context.Context, r *http.Request) error {
		f, err := temp(r)
		if err != nil {
			return err
		}
		return s.Grill.SetGrillTemp(ctx, f)
	}))
	mux.HandleFunc("PUT /api/temperature/food/{tempF}", s.command(func(ctx context.Context, r *http.Request) error {
		f, err := temp(r)
		if err != nil {
			return err
		}
		return s.Grill.SetFoodTemp(ctx, f)
	}))
	mux.HandleFunc("GET /api/temperature/history", s.history)
	mux.HandleFunc("GET /api/connection", s.connection)
	mux.HandleFunc("GET /api/events", s.events)
	mux.HandleFunc("GET /api", apiDocs)
	mux.HandleFunc("GET /api/{$}", apiDocs)
	mux.HandleFunc("/", s.static) // also turns unknown routes and wrong methods into 404
	return logRequests(s.Log, cors(s.AllowedOrigins, mux))
}

//go:embed docs.html
var docsHTML []byte

// apiDocs serves the API documentation page.
func apiDocs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(docsHTML)
}

func temp(r *http.Request) (int, error) {
	v := r.PathValue("tempF")
	f, err := strconv.Atoi(v)
	if err != nil {
		return 0, &grill.InvalidCommandError{Msg: "Invalid temperature: " + v}
	}
	return f, nil
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	st, err := s.Grill.Status(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, st)
}

func (s *Server) command(fn func(context.Context, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(r.Context(), r); err != nil {
			s.fail(w, err)
			return
		}
		writeText(w, http.StatusOK, "OK")
	}
}

func (s *Server) connection(w http.ResponseWriter, _ *http.Request) {
	if s.Health == nil {
		notFound(w)
		return
	}
	writeJSON(w, s.Health())
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	since, err := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	if err != nil { // the Node server returned no rows for a missing/invalid since
		writeJSON(w, []store.Row{})
		return
	}
	rows, err := s.History.History(r.Context(), since)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, rows)
}

// events streams `status`, `connection` and `alert` events (Server-Sent Events).
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	c := s.Hub.subscribe()
	defer s.Hub.unsubscribe(c)

	w.Write([]byte("retry: 3000\n\n"))
	if s.LastStatus != nil {
		if st, ok := s.LastStatus(); ok {
			if msg, err := formatEvent("status", st); err == nil {
				w.Write(msg)
			}
		}
	}
	if s.Health != nil {
		if msg, err := formatEvent("connection", s.Health()); err == nil {
			w.Write(msg)
		}
	}
	if err := rc.Flush(); err != nil {
		return
	}
	keepAlive := s.KeepAlive
	if keepAlive <= 0 {
		keepAlive = 25 * time.Second
	}
	t := time.NewTicker(keepAlive)
	defer t.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-c:
			w.Write(msg)
		case <-t.C:
			w.Write([]byte(": keep-alive\n\n"))
		}
		if err := rc.Flush(); err != nil {
			return
		}
	}
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	if s.Static == nil || strings.HasPrefix(r.URL.Path, "/api/") || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		notFound(w)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	if f, err := fs.Stat(s.Static, name); err != nil || f.IsDir() {
		notFound(w)
		return
	}
	if strings.HasPrefix(name, "assets/") { // Vite output has content-hashed names
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	http.ServeFileFS(w, r, s.Static, name)
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	var inv *grill.InvalidCommandError
	code := http.StatusInternalServerError
	if errors.As(err, &inv) {
		code = http.StatusBadRequest
	} else {
		s.Log.Error("request failed", "err", err)
	}
	writeText(w, code, err.Error())
}

func notFound(w http.ResponseWriter) { writeText(w, http.StatusNotFound, "Not Found") }

// writeText writes a plain-text body without the trailing newline that
// http.Error adds, matching the original server's responses.
func writeText(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func writeJSON(w http.ResponseWriter, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		writeText(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(b)
}

// cors lets browsers on the allowed origins call the API, and refuses
// commands from any other site. Without it, any web page opened by someone
// on the LAN could turn the grill on. Requests without an Origin header
// (curl, scripts) are not affected.
func cors(allowed []string, next http.Handler) http.Handler {
	anyOrigin := slices.Contains(allowed, "*")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		safe := r.Method == http.MethodGet || r.Method == http.MethodHead
		if origin == "" {
			// Browsers send Origin with every PUT; this covers old ones.
			if !safe && !anyOrigin && r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				writeText(w, http.StatusForbidden, "Cross-origin request refused")
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if sameOrigin(origin, r.Host) {
			next.ServeHTTP(w, r)
			return
		}
		if !anyOrigin && !slices.Contains(allowed, origin) {
			if !safe { // includes preflights
				writeText(w, http.StatusForbidden, "Cross-origin request refused: "+origin+" is not in GMG_CORS_ORIGINS")
				return
			}
			next.ServeHTTP(w, r) // no CORS headers, so the browser won't expose the response
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Add("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,PUT,POST,DELETE")
			w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// sameOrigin reports whether origin (scheme://host[:port]) names the host
// the request was sent to.
func sameOrigin(origin, host string) bool {
	u, err := url.Parse(origin)
	return err == nil && u.Host != "" && strings.EqualFold(u.Host, host)
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(code int)        { s.code = code; s.ResponseWriter.WriteHeader(code) }
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func logRequests(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Debug("http", "method", r.Method, "path", r.URL.Path, "status", rec.code, "dur", time.Since(start))
	})
}
