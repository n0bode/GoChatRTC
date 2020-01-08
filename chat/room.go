package chat

import (
	"sync"

	"nhooyr.io/websocket"
)

type Room struct {
	conns map[string]*websocket.Conn
	m     *sync.RWMutex
}

func (r *Room) Store(userID string, conn *websocket.Conn) {
	r.m.Lock()
	defer r.m.Unlock()
	r.conns[userID] = conn
}

func (r *Room) Load(userID string) (conn *websocket.Conn, ok bool) {
	r.m.RLock()
	defer r.m.RUnlock()
	conn, ok = r.conns[userID]
	return
}

func (r *Room) Delete(userID string) {
	r.m.Lock()
	defer r.m.Unlock()
	delete(r.conns, userID)
}

func (r *Room) Range(rfunc func(userID string, conn *websocket.Conn)) {
	r.m.RLock()
	defer r.m.RUnlock()
	for key, conn := range r.conns {
		rfunc(key, conn)
	}
}

func (r *Room) Length() int {
	r.m.RLock()
	defer r.m.RUnlock()
	return len(r.conns)
}

func NewRoom() *Room {
	return &Room{
		conns: make(map[string]*websocket.Conn),
		m:     &sync.RWMutex{},
	}
}
