// Command management-plane serves:
//   - /_mgmt/*  — provisioning/admin API, docs/contracts/management-plane-openapi.yaml
//   - /login, /logout, and everything else — the single-domain session broker
//     and reverse proxy from docs/adr/0008-single-domain-identity-routing.md
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

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	mgmtAPIKey := os.Getenv("MGMT_API_KEY")
	if mgmtAPIKey == "" {
		log.Println("WARNING: MGMT_API_KEY not set — /_mgmt auth middleware disabled, do not run this outside local dev like this")
	}

	r := gin.Default()
	store := tenant.NewMemStore()

	// --- /_mgmt: provisioning/admin API, unchanged from before this ADR ---
	if mgmtAPIKey != "" {
		mgmt := r.Group("/_mgmt")
		mgmt.Use(func(c *gin.Context) {
			if c.GetHeader("X-Mgmt-Api-Key") != mgmtAPIKey {
				c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
				return
			}
			c.Next()
		})
	}
	h := &api.Handler{Store: store}
	api.RegisterRoutes(r, h)

	// --- /login, /logout, and the tenant reverse proxy (docs/adr/0008) ---
	sessions := auth.NewStore()
	authHandler := auth.NewHandler(sessions, store)
	r.POST("/login", authHandler.Login)
	r.POST("/logout", authHandler.Logout)
	r.NoRoute(authHandler.RequireSession, authHandler.ProxyToTenant)

	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })

	log.Printf("management-plane listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
