// Command management-plane serves docs/contracts/management-plane-openapi.yaml.
//
// Phase 2 status: routes are wired and provisioning is a real allocation
// (slug + network per docs/adr/0004), but nothing actually deploys to k8s
// yet — createTenant reserves state only. See TODOs in internal/api and
// internal/k8s.
package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/nexwall/nexwall-multi-tenant/management-plane/internal/api"
	"github.com/nexwall/nexwall-multi-tenant/management-plane/internal/tenant"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	apiKey := os.Getenv("MGMT_API_KEY")
	if apiKey == "" {
		log.Println("WARNING: MGMT_API_KEY not set — auth middleware disabled, do not run this outside local dev like this")
	}

	r := gin.Default()

	if apiKey != "" {
		r.Use(func(c *gin.Context) {
			if c.GetHeader("X-Mgmt-Api-Key") != apiKey {
				c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
				return
			}
			c.Next()
		})
	}

	store := tenant.NewMemStore()
	h := &api.Handler{Store: store}
	api.RegisterRoutes(r, h)

	r.GET("/healthz", func(c *gin.Context) { c.Status(200) })

	log.Printf("management-plane listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
