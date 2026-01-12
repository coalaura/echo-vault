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
)

type Event struct {
	Type int    `json:"type"`
	Hash string `json:"hash,omitempty"`
	Echo *Echo  `json:"echo,omitempty"`

	Size  uint64 `json:"size"`
	Count uint64 `json:"count"`
}

type Client chan []byte

type Hub struct {
	clients map[*Client]struct{}

	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),

		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.clients[client] = struct{}{}
		case client := <-h.unregister:
			delete(h.clients, client)
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case *client <- message:
				default:
					// drop message if client buffer is full
				}
			}
		}
	}
}

func (h *Hub) Broadcast(event Event) {
	event.Size = usage.Load()
	event.Count = count.Load()

	b, err := json.Marshal(event)
	if err != nil {
		log.Warnf("Failed to encode event: %v\n", err)

		return
	}

	h.broadcast <- b
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

	client := make(Client, 10)

	h.register <- &client

	defer func() {
		h.unregister <- &client

		close(client)
	}()

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-client:
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
