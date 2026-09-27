package tenant

import (
	"errors"
	"strconv"
	"sync"
	"time"
)

// MemStore is a throwaway Store implementation so the API contract can be
// exercised end-to-end before Phase 2's real Postgres-backed store exists.
// Not safe to use beyond local dev — state is lost on restart.
type MemStore struct {
	mu      sync.Mutex
	nextID  int
	tenants map[int]*Tenant
	slugs   map[string]bool
}

func NewMemStore() *MemStore {
	return &MemStore{
		nextID:  1,
		tenants: make(map[int]*Tenant),
		slugs:   make(map[string]bool),
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

func (s *MemStore) Create(req CreateRequest) (*Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.slugs[req.Slug] {
		return nil, errors.New("slug already taken")
	}
	id := s.nextID
	s.nextID++
	cidr, port := AllocateNetwork(id)
	t := &Tenant{
		TenantID:    id,
		Slug:        req.Slug,
		DisplayName: req.DisplayName,
		Plan:        req.Plan,
		Status:      StatusProvisioning,
		CreatedAt:   time.Now().UTC(),
		VPNCidr:     cidr,
		VPNPort:     port,
		Subdomain:   req.Slug + ".painel.nexwall.com.br",
		Namespace:   "tenant-" + strconv.Itoa(id) + "-" + req.Slug,
		HelmRelease: "tenant-" + strconv.Itoa(id) + "-" + req.Slug,
	}
	s.tenants[id] = t
	s.slugs[req.Slug] = true
	cp := *t
	return &cp, nil
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
