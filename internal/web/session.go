package web

import (
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	sessionName     = "hortbot-session-v1"
	sessionTwitchID = "twitch_id"
	sessionUsername = "twitch_username"
)

type session struct {
	s *sessions.Session
}

func (a *App) getSession(r *http.Request) *session {
	s, _ := a.store.Get(r, sessionName)
	return &session{s: s}
}

func (a *App) clearSession(w http.ResponseWriter, r *http.Request) error {
	s, _ := a.store.Get(r, sessionName)
	s.Options.MaxAge = -1
	if err := s.Save(r, w); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}
	return nil
}

func (s *session) save(w http.ResponseWriter, r *http.Request) error {
	if err := s.s.Save(r, w); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}
	return nil
}

func (s *session) clearValues() {
	s.s.Values = make(map[any]any)
}

func (s *session) setTwitchID(id int64) {
	s.s.Values[sessionTwitchID] = id
}

// func (s *session) getTwitchID() int64 {
// 	v, _ := s.s.Values[sessionTwitchID].(int64)
// 	return v
// }

func (s *session) setUsername(name string) {
	s.s.Values[sessionUsername] = name
}

func (s *session) getUsername() string {
	v, _ := s.s.Values[sessionUsername].(string)
	return v
}
