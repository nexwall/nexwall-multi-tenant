// Package api implements docs/contracts/management-plane-openapi.yaml.
// Handlers are intentionally thin — all real logic belongs in internal/tenant
// and internal/k8s so it stays testable without an HTTP layer.
package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/nexwall/nexwall-multi-tenant/management-plane/internal/tenant"
)

// Store is the minimal persistence interface handlers need. A real
// implementation (Postgres, per ADR 0003) lands in Phase 2; an in-memory
// implementation is enough to exercise the contract end-to-end today.
type Store interface {
	List() ([]tenant.Tenant, error)
	Get(id int) (*tenant.Tenant, error)
	Create(req tenant.CreateRequest) (*tenant.Tenant, error)
	Delete(id int) error
	SetStatus(id int, status tenant.Status) error
	HandoffSecret(id int) (string, error)
}

type Handler struct {
	Store Store
}

// RegisterRoutes mounts the admin/provisioning API onto a group the caller
// has already attached auth middleware to (see main.go) — this function
// must never call r.Group itself. An earlier version did exactly that,
// creating a second, middleware-less "/_mgmt" group that silently shadowed
// nothing and left every route below completely unauthenticated despite
// MGMT_API_KEY appearing to be configured. Found during the ADR 0009 work,
// fixed here; see the commit message for how it was found.
func RegisterRoutes(g *gin.RouterGroup, h *Handler) {
	g.GET("/tenants", h.listTenants)
	g.POST("/tenants", h.createTenant)
	g.GET("/tenants/:id", h.getTenant)
	g.DELETE("/tenants/:id", h.deleteTenant)
	g.GET("/tenants/:id/status", h.tenantStatus)
	g.POST("/tenants/:id/suspend", h.suspendTenant)
	g.POST("/tenants/:id/resume", h.resumeTenant)
}

// RegisterPartnerRoutes mounts the one endpoint nexwall-partner-multitenant
// is allowed to call, on a group gated by PARTNER_API_KEY specifically --
// deliberately not reachable via MGMT_API_KEY, and not part of
// RegisterRoutes above. See docs/adr/0009-handoff-secret-provisioning.md.
func RegisterPartnerRoutes(g *gin.RouterGroup, h *Handler) {
	g.GET("/tenants/:id/handoff-secret", h.handoffSecret)
}

func (h *Handler) listTenants(c *gin.Context) {
	tenants, err := h.Store.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tenants)
}

func (h *Handler) createTenant(c *gin.Context) {
	var req tenant.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.Store.Create(req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	// TODO(phase-2): actually provision — call k8s.CreateTenantNamespace then
	// k8s.InstallOrUpgradeRelease with values rendered from t. Today this
	// only reserves the slug/network allocation; nothing is deployed yet.
	c.JSON(http.StatusCreated, t)
}

func (h *Handler) getTenant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	t, err := h.Store.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *Handler) deleteTenant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	// TODO(phase-2): trigger namespace deletion via k8s client before
	// removing from the store — deleting the store row first would orphan
	// a running namespace.
	if err := h.Store.Delete(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) tenantStatus(c *gin.Context) {
	// TODO(phase-2): query k8s.GetTenantStatus(namespace), not the store.
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented, see docs/phases/phase-2-management-plane-mvp.md"})
}

func (h *Handler) suspendTenant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	// TODO(phase-2): scale the tenant's Deployments to 0 via k8s client
	// before flipping status — status must reflect reality, not intent.
	if err := h.Store.SetStatus(id, tenant.StatusSuspended); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	t, _ := h.Store.Get(id)
	c.JSON(http.StatusOK, t)
}

func (h *Handler) resumeTenant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.Store.SetStatus(id, tenant.StatusActive); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	t, _ := h.Store.Get(id)
	c.JSON(http.StatusOK, t)
}

func (h *Handler) handoffSecret(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	secret, err := h.Store.HandoffSecret(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tenant_id": id, "handoff_secret": secret})
}

func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return 0, false
	}
	return id, true
}
