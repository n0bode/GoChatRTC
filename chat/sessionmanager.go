package chat

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

type SessionManager struct {
	m        *sync.RWMutex
	ark      string
	sessions map[string]Session
}

// Store a new Session
func (sm *SessionManager) Store(expire time.Duration, userID string) (token string) {
	var future time.Time = time.Now().Add(expire)
	token = EncodeToSha(sm.ark + userID)
	sm.m.Lock()
	sm.sessions[token] = Session{
		Expire: future,
		UserID: userID,
	}
	sm.m.Unlock()
	return
}

// IsValid check if a token is valid
// Return a
func (sm *SessionManager) IsValid(token string) (auth Session, valid bool) {
	sm.m.RLock()
	if auth, valid = sm.sessions[token]; valid {
		if valid = (auth.Expire.Sub(time.Now()) >= 0); !valid {
			delete(sm.sessions, token)
		}
	}
	sm.m.RUnlock()
	return
}

// Middleware is used to before restrict endpoint
func (sm *SessionManager) Middleware(sucess http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if len(token) == 0 {
			w.WriteHeader(http.StatusNonAuthoritativeInfo)
			return
		}

		session, valid := sm.IsValid(token)
		if !valid {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		r.Header.Add("userID", session.UserID)
		sucess.ServeHTTP(w, r)
	})
}

// NewSessionManager create a manager for session
func NewSessionManager() *SessionManager {
	val := make([]byte, 64)
	rand.Read(val)

	return &SessionManager{
		ark:      hex.EncodeToString(val),
		sessions: make(map[string]Session),
		m:        &sync.RWMutex{},
	}
}
