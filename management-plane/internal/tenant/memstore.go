package tenant

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nexwall/nexwall-multitenant-sso/handoff"
)

// MemStore is a throwaway Store implementation so the API contract can be
// exercised end-to-end before Phase 2's real Postgres-backed store exists.
// Not safe to use beyond local dev — state is lost on restart.
type MemStore struct {
	mu         sync.Mutex
	nextID     int
	tenants    map[int]*Tenant
	slugs      map[string]bool
	emailIndex map[string][]int // lowercased email -> tenant ids
}

func NewMemStore() *MemStore {
	return &MemStore{
		nextID:     1,
		tenants:    make(map[int]*Tenant),
		slugs:      make(map[string]bool),
		emailIndex: make(map[string][]int),
	}
}

func (s *MemStore) List() ([]Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		out = append(out, *t)
	}
	return out, nil
}

func (s *MemStore) Get(id int) (*Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *t
	return &cp, nil
}

// GetBySlug is what GET /sso/handoff (docs/adr/0009) looks the target
// tenant up by -- the partner portal knows a tenant's slug, not its
// internal integer id.
func (s *MemStore) GetBySlug(slug string) (*Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tenants {
		if t.Slug == slug {
			cp := *t
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (s *MemStore) Create(req CreateRequest) (*Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.slugs[req.Slug] {
		return nil, errors.New("slug already taken")
	}
	id := s.nextID
	s.nextID++
	cidr, port := AllocateNetwork(id)
	handoffSecret, err := handoff.GenerateSecret()
	if err != nil {
		return nil, err
	}
	t := &Tenant{
		AdminEmail:    req.AdminEmail,
		HandoffSecret: handoffSecret,
		TenantID:      id,
		Slug:          req.Slug,
		DisplayName:   req.DisplayName,
		Plan:          req.Plan,
		Status:        StatusProvisioning,
		CreatedAt:     time.Now().UTC(),
		VPNCidr:       cidr,
		VPNPort:       port,
		Subdomain:     req.Slug + ".painel.nexwall.com.br",
		Namespace:     "tenant-" + strconv.Itoa(id) + "-" + req.Slug,
		HelmRelease:   "tenant-" + strconv.Itoa(id) + "-" + req.Slug,
	}
	s.tenants[id] = t
	s.slugs[req.Slug] = true
	if req.AdminEmail != "" {
		email := strings.ToLower(req.AdminEmail)
		s.emailIndex[email] = append(s.emailIndex[email], id)
	}
	cp := *t
	return &cp, nil
}

// TenantsForEmail returns the tenant IDs an email is known to belong to.
// Today this is only populated at provisioning time (the tenant's initial
// admin_email) — see the "open item" in docs/adr/0008 about users created
// later inside a tenant not yet syncing back here.
func (s *MemStore) TenantsForEmail(email string) []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int(nil), s.emailIndex[strings.ToLower(email)]...)
}

// HandoffSecret returns the raw secret for one tenant. The only caller
// should be the GET /_mgmt/tenants/:id/handoff-secret handler, gated by
// PARTNER_API_KEY -- see docs/adr/0009-handoff-secret-provisioning.md.
func (s *MemStore) HandoffSecret(id int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[id]
	if !ok {
		return "", errors.New("not found")
	}
	return t.HandoffSecret, nil
}

func (s *MemStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[id]
	if !ok {
		return errors.New("not found")
	}
	delete(s.slugs, t.Slug)
	delete(s.tenants, id)
	return nil
}

func (s *MemStore) SetStatus(id int, status Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[id]
	if !ok {
		return errors.New("not found")
	}
	t.Status = status
	return nil
}
