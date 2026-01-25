package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const (
	EventCreateEcho = iota
	EventUpdateEcho
	EventDeleteEcho
	EventProcessingEcho
)

type Event struct {
	Type       int    `json:"type"`
	ID         string `json:"id,omitempty"`
	Hash       string `json:"hash,omitempty"`
	Echo       *Echo  `json:"echo,omitempty"`
	Processing bool   `json:"processing,omitempty"`

	Size  uint64 `json:"size"`
	Count uint64 `json:"count"`
}

type Client struct {
	send chan []byte
}

type Hub struct {
	clients map[*Client]struct{}

	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client

	ctx context.Context
}

func NewHub(ctx context.Context) *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),

		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),

		ctx: ctx,
	}
}

func (h *Hub) Run() {
	defer h.closeAll()

	for {
		select {
		case <-h.ctx.Done():
			return

		case client := <-h.register:
			h.clients[client] = struct{}{}

		case client := <-h.unregister:
			h.removeClient(client)

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					h.removeClient(client)
				}
			}
		}
	}
}

func (h *Hub) closeAll() {
	for client := range h.clients {
		close(client.send)
		delete(h.clients, client)
	}
}

func (h *Hub) removeClient(c *Client) {
	if _, ok := h.clients[c]; !ok {
		return
	}

	delete(h.clients, c)
	close(c.send)
}

func (h *Hub) Broadcast(event Event) {
	event.Size = usage.Load()
	event.Count = count.Load()

	b, err := json.Marshal(event)
	if err != nil {
		log.Warnf("Failed to encode event: %v\n", err)

		return
	}

	select {
	case <-h.ctx.Done():
		return
	case h.broadcast <- b:
	}
}

func (h *Hub) BroadcastCreate(id string, echo *Echo) {
	h.Broadcast(Event{
		Type: EventCreateEcho,
		ID:   id,
		Echo: echo,
	})
}

func (h *Hub) BroadcastUpdate(echo *Echo) {
	h.Broadcast(Event{
		Type: EventUpdateEcho,
		Echo: echo,
	})
}

func (h *Hub) BroadcastDelete(hash string) {
	h.Broadcast(Event{
		Type: EventDeleteEcho,
		Hash: hash,
	})
}

func (h *Hub) BroadcastProcessing(hash string, processing bool) {
	h.Broadcast(Event{
		Type:       EventProcessingEcho,
		Hash:       hash,
		Processing: processing,
	})
}

func (h *Hub) Handle(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		abort(w, http.StatusInternalServerError, "streaming unsupported")

		return
	}

	rc := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	client := &Client{
		send: make(chan []byte, 8),
	}

	select {
	case <-h.ctx.Done():
		return
	case h.register <- client:
	}

	defer func() {
		select {
		case <-h.ctx.Done():
		case h.unregister <- client:
		}
	}()

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-client.send:
			if !ok {
				return
			}

			if err := h.writeSSE(w, rc, flusher, msg); err != nil {
				return
			}
		case <-heartbeat.C:
			if err := h.writeSSE(w, rc, flusher, []byte("ping")); err != nil {
				return
			}
		}
	}
}

func (h *Hub) writeSSE(w http.ResponseWriter, rc *http.ResponseController, flusher http.Flusher, data []byte) error {
	err := rc.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte("\n"))
	if err != nil {
		return err
	}

	flusher.Flush()

	return nil
}
