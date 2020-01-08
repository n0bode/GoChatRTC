package chat

import "sync"

type RoomManager struct {
	rooms map[string]*Room
	m     *sync.RWMutex
}

func (rm *RoomManager) Store(roomID string) (room *Room) {
	rm.m.Lock()
	defer rm.m.Unlock()

	if _, ok := rm.rooms[roomID]; !ok {
		room = NewRoom()
		rm.rooms[roomID] = room
	}
	return
}

func (rm *RoomManager) Load(roomID string) (room *Room, ok bool) {
	rm.m.RLock()
	defer rm.m.RUnlock()
	room, ok = rm.rooms[roomID]
	return
}

func (rm *RoomManager) Delete(roomID string) {
	rm.m.Lock()
	defer rm.m.Unlock()
	delete(rm.rooms, roomID)
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
		m:     &sync.RWMutex{},
	}
}
