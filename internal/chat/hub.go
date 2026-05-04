package chat

import (
	"log/slog"
	"sync"
)

type Message struct {
	Server          string `json:"server"`
	Sender          string `json:"sender"`
	SenderFormatted string `json:"senderFormatted"`
	Text            string `json:"text"`
}

type Hub struct {
	m  map[string]chan Message
	mu sync.RWMutex

	in chan Message
}

func NewHub() *Hub {
	h := &Hub{
		m:  make(map[string]chan Message),
		mu: sync.RWMutex{},
		in: make(chan Message, 100),
	}

	go h.fanout()

	return h
}

func (h *Hub) fanout() {
	for msg := range h.in {
		h.mu.RLock()
		for id, v := range h.m {
			if msg.Server == id {
				continue
			}
			select {
			case v <- msg:
			default:
				// TODO: notify
				slog.Warn("Channel overflow", slog.String("id", id))
			}
		}
		h.mu.RUnlock()
	}
}

// Register channel for broadcast (no-op if nil)
func (h *Hub) Register(id string, ch chan Message) {
	if ch != nil {
		h.mu.Lock()
		h.m[id] = ch
		h.mu.Unlock()
		slog.Info("Channel registered", slog.String("id", id))
	}
}

func (h *Hub) Send(msg Message) {
	h.in <- msg
	slog.Debug("Message for broadcast",
		slog.String("server", msg.Server),
		slog.String("sender", msg.Sender),
		slog.String("text", msg.Text))
}

// Close channel and remove from broadcast
func (h *Hub) Unregister(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.m[id]; ok {
		close(c)
		delete(h.m, id)
		slog.Info("Channel removed", slog.String("id", id))
	}
}

func (h *Hub) Close() {
	close(h.in)
	h.mu.Lock()
	defer h.mu.Unlock()
	for k, c := range h.m {
		close(c)
		delete(h.m, k)
	}
	slog.Info("All hub channels closed")
}
