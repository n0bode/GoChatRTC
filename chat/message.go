package chat

type Message struct {
	ID    uint64 `rethinkdb:"id"`
	Hash  string `rethinkdb:"hash"`
	Owner string `rethinkdb:"owner"`
	Room  string `rethinkdb:"room"`
	Text  string `rethinkdb:"text"`
	Time  int64  `rethindb:"time"`
}
