// Package auth implements the login broker and session store behind
// docs/adr/0008-single-domain-identity-routing.md: the Management Plane
// never stores a password, it replays the submitted credentials against the
// owning tenant's own /api/login and, on success, keeps that tenant's JWT
// server-side, keyed by an opaque session cookie.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const (
	CookieName = "nx_session"
	sessionTTL = 8 * time.Hour
)

// Session is what a logged-in browser is holding a cookie for. TenantID is
// which tenant's Service the reverse proxy forwards to; TenantJWT is what
// gets injected as the Authorization header on every proxied request.
type Session struct {
	ID        string
	Email     string
	TenantID  int
	TenantJWT string
	ExpiresAt time.Time
}

// Store is an in-memory session store. Fine for a single Management Plane
// replica (see docs/adr/0008's consequences) — revisit before running more
// than one.
type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

func NewStore() *Store {
	st := &Store{sessions: make(map[string]*Session)}
	go st.reapExpiredLoop()
	return st
}

func (s *Store) Create(email string, tenantID int, tenantJWT string) (*Session, error) {
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	sess := &Session{
		ID:        id,
		Email:     email,
		TenantID:  tenantID,
		TenantJWT: tenantJWT,
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return sess, nil
}

func (s *Store) Get(id string) (*Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	return sess, true
}

// SwitchTenant replaces which tenant a session targets, e.g. after a
// multi-tenant account picks a different org — no re-login needed since the
// original credentials already proved ownership of that tenant too (see
// Handler.pickTenant in http.go).
func (s *Store) SwitchTenant(id string, tenantID int, tenantJWT string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[id]; ok {
		sess.TenantID = tenantID
		sess.TenantJWT = tenantJWT
	}
}

func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func (s *Store) reapExpiredLoop() {
	for range time.Tick(5 * time.Minute) {
		now := time.Now()
		s.mu.Lock()
		for id, sess := range s.sessions {
			if now.After(sess.ExpiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
