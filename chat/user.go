package chat

// User is a struct to know about user
type User struct {
	ID        string   `rethinkdb:"id,omitempty" json:"-"`
	Name      string   `rethinkdb:"name" json:"name"`
	UserID    string   `rethinkdb:"userID" json:"userID"`
	SecretKey string   `rethinkdb:"secretKey" json:"secretKey,omitempty"`
	Created   string   `rethinkdb:"created" json:"created"`
	LastTime  string   `rethinkdb:"lasttime" json:"lastTime,omitempty"`
	Invites   []Invite `rethinkdb:"invites" json:"invites,omitempty"`
	Friends   []string `rethinkdb:"friends" json:"friends,omitempty"`
}
