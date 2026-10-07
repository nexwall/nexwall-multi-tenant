// Command management-plane serves:
//   - /_mgmt/*  — provisioning/admin API (MGMT_API_KEY), docs/contracts/management-plane-openapi.yaml
//   - /_mgmt/tenants/:id/handoff-secret — the one endpoint nexwall-partner-multitenant
//     calls (PARTNER_API_KEY, deliberately separate from MGMT_API_KEY), docs/adr/0009
//   - /login, /logout, /sso/handoff, and everything else — the single-domain
//     session broker and reverse proxy, docs/adr/0008 and docs/adr/0009
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/nexwall/nexwall-multi-tenant/management-plane/internal/api"
	"github.com/nexwall/nexwall-multi-tenant/management-plane/internal/auth"
	"github.com/nexwall/nexwall-multi-tenant/management-plane/internal/tenant"
)

func apiKeyMiddleware(header, expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader(header) != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	mgmtAPIKey := os.Getenv("MGMT_API_KEY")
	partnerAPIKey := os.Getenv("PARTNER_API_KEY")
	if mgmtAPIKey == "" {
		log.Println("WARNING: MGMT_API_KEY not set — /_mgmt provisioning routes are NOT mounted at all (see below), nothing to disable")
	}
	if partnerAPIKey == "" {
		log.Println("WARNING: PARTNER_API_KEY not set — the handoff-secret endpoint is NOT mounted at all")
	}

	r := gin.Default()
	store := tenant.NewMemStore()
	h := &api.Handler{Store: store}

	// --- /_mgmt: provisioning/admin API ---
	// Routes are only mounted when their key is configured -- unlike the
	// pre-ADR-0009 version of this file, there is no "middleware disabled"
	// fallback that leaves the routes reachable unauthenticated. See the
	// comment on api.RegisterRoutes for why this matters: a prior version
	// of this file created a second, unprotected group at the same path
	// without anyone noticing, because the routes still worked in every
	// manual test (an API key was always sent, so a missing *check* was
	// never exercised).
	if mgmtAPIKey != "" {
		mgmt := r.Group("/_mgmt", apiKeyMiddleware("X-Mgmt-Api-Key", mgmtAPIKey))
		api.RegisterRoutes(mgmt, h)
	}
	if partnerAPIKey != "" {
		partner := r.Group("/_mgmt", apiKeyMiddleware("X-Partner-Api-Key", partnerAPIKey))
		api.RegisterPartnerRoutes(partner, h)
	}

	// --- /login, /logout, /sso/handoff, and the tenant reverse proxy ---
	sessions := auth.NewStore()
	authHandler := auth.NewHandler(sessions, store)
	r.POST("/login", authHandler.Login)
	r.POST("/logout", authHandler.Logout)
	r.GET("/sso/handoff", authHandler.Handoff)
	r.NoRoute(authHandler.RequireSession, authHandler.ProxyToTenant)

	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })

	log.Printf("management-plane listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
