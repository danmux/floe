package server

import (
	"crypto/rand"
	"fmt"
	"time"
)

const seshLifetime = time.Hour * 240 // 10 days

type session struct {
	token      string
	lastActive time.Time
	user       string
}

var tokens = map[string]*session{}

func goodToken(token string) *session {
	t, ok := tokens[token]
	if !ok {
		return nil
	}

	// expired?
	now := time.Now()
	d := now.Sub(t.lastActive)
	if d > seshLifetime {
		return nil
	}

	// update last active in token store
	t.lastActive = now
	t.token = token

	return t
}

// TODO - inject an authenticator
func login(user, pass string) string {
	if user == "admin" && pass == "password" {
		b := make([]byte, 8)
		rand.Read(b)
		token := fmt.Sprintf("%x", b)
		tokens[token] = &session{
			lastActive: time.Now(),
			user:       user,
		}
		return token
	}
	return ""
}

func logout(token string) {
	delete(tokens, token)
}
