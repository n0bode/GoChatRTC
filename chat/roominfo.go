package chat

type RoomInfo struct {
	ID       string   `rethinkdb:"id"`
	Name     string   `rethinkdb:"name"`
	Password string   `rethinkdb:"password"`
	Peers    []string `rethinkdb:"peers"`
}
