package chat

import (
	"context"
	"errors"
	"sync"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Hub struct {
	peers map[string]*websocket.Conn
	m     *sync.RWMutex
}

func (h *Hub) Add(userID string, conn *websocket.Conn) {
	h.m.Lock()
	defer h.m.Unlock()
	h.peers[userID] = conn
}

func (h *Hub) Rem(userID string) {
	h.m.Lock()
	defer h.m.Unlock()
	if _, ok := h.peers[userID]; ok {
		delete(h.peers, userID)
	}
}

func (h *Hub) WriteTo(ctx context.Context, event string, message interface{}, to string) (err error) {
	h.m.RLock()
	defer h.m.RUnlock()
	if peer, ok := h.peers[to]; ok {
		err = wsjson.Write(ctx, peer, map[string]interface{}{
			"event":   event,
			"message": message,
		})
		return
	}
	return errors.New("Peer not found")
}

func (h *Hub) Write(ctx context.Context, event string, message interface{}) {
	h.m.RLock()
	defer h.m.RUnlock()
	for _, peer := range h.peers {
		wsjson.Write(ctx, peer, map[string]interface{}{
			"event":   event,
			"message": message,
		})
	}
}

func NewHub() *Hub {
	return &Hub{
		m:     &sync.RWMutex{},
		peers: make(map[string]*websocket.Conn),
	}
}
