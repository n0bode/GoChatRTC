package chat

type Invite struct {
	From   string `rethinkdb:"from" json:"from"`
	To     string `rethinkdb:"to" json:"to"`
	RoomID string `rethinkdb:"roomID" json:"roomID"`
}
