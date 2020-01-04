package chat

type Room struct {
	ID       string   `rethinkdb:"id"`
	Name     string   `rethinkdb:"name"`
	Password string   `rethinkdb:"password"`
	Users    []string `rethinkdb:"users"`
}
