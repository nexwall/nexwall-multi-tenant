// Package tenant defines the Management Plane's own data model.
// This is deliberately separate from nexwall-controller's models — see
// docs/adr/0003-management-plane-as-separate-service.md. Nothing here is
// shared with any tenant's TimescaleDB.
package tenant

import (
	"strconv"
	"time"
)

type Status string

const (
	StatusProvisioning Status = "provisioning"
	StatusActive       Status = "active"
	StatusSuspended    Status = "suspended"
	StatusDeleting     Status = "deleting"
)

type Plan string

const (
	PlanTrial      Plan = "trial"
	PlanStandard   Plan = "standard"
	PlanEnterprise Plan = "enterprise"
)

// Tenant mirrors the OpenAPI "Tenant" schema in
// docs/contracts/management-plane-openapi.yaml. Keep these in sync by hand
// until the contract is code-generated (tracked for Phase 2).
type Tenant struct {
	TenantID    int       `json:"tenant_id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"display_name"`
	Plan        Plan      `json:"plan"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`

	// Allocated per docs/adr/0004-network-allocation.md at creation time,
	// never recomputed, never reused after deletion.
	VPNCidr     string `json:"vpn_cidr"`
	VPNPort     int    `json:"vpn_port"`
	Subdomain   string `json:"subdomain"`
	Namespace   string `json:"namespace"`
	HelmRelease string `json:"helm_release"`
}

// CreateRequest mirrors the OpenAPI "TenantCreateRequest" schema.
type CreateRequest struct {
	Slug        string `json:"slug" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Plan        Plan   `json:"plan" binding:"required"`
	AdminEmail  string `json:"admin_email"`
}

// AllocateNetwork implements the scheme from ADR 0004. tenantID must already
// be assigned (auto-increment from the store) before calling this.
func AllocateNetwork(tenantID int) (cidr string, vpnPort int) {
	cidr = fmtCIDR(100 + tenantID)
	vpnPort = 20000 + tenantID
	return
}

func fmtCIDR(octet int) string {
	return "10." + strconv.Itoa(octet) + ".0.0/16"
}
