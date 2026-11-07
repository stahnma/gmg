package api

import (
	"encoding/json"
	"sync"
)

// Hub fans out server-sent events to connected browsers.
type Hub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

// NewHub returns an empty hub.
func NewHub() *Hub { return &Hub{clients: map[chan []byte]struct{}{}} }

// Publish sends an event to every connected client. Slow clients that have
// fallen behind miss events rather than blocking the poller.
func (h *Hub) Publish(event string, payload any) {
	msg, err := formatEvent(event, payload)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c <- msg:
		default:
		}
	}
}

// Clients returns the number of connected clients.
func (h *Hub) Clients() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *Hub) subscribe() chan []byte {
	c := make(chan []byte, 16)
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	return c
}

func (h *Hub) unsubscribe(c chan []byte) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func formatEvent(event string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return []byte("event: " + event + "\ndata: " + string(data) + "\n\n"), nil
}
