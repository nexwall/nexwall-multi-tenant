package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nexwall/nexwall-multi-tenant/management-plane/internal/tenant"
)

// TenantLookup is the subset of tenant.Store the auth Handler needs. Kept as
// an interface so tests don't need a real Store implementation.
type TenantLookup interface {
	TenantsForEmail(email string) []int
	Get(id int) (*tenant.Tenant, error)
	GetBySlug(slug string) (*tenant.Tenant, error)
}

type Handler struct {
	Sessions *Store
	Tenants  TenantLookup
	// HTTPClient talks to tenants' in-cluster Services. Short timeout: a
	// login call that hangs should fail fast, not hold the browser forever.
	HTTPClient *http.Client
}

func NewHandler(sessions *Store, tenants TenantLookup) *Handler {
	return &Handler{
		Sessions:   sessions,
		Tenants:    tenants,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	// TenantID: the browser sends this on the *second* call, after the user
	// picked one from the list the first call returned (see pickTenant below).
	TenantID int `json:"tenant_id,omitempty"`
}

type tenantChoice struct {
	TenantID    int    `json:"tenant_id"`
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
}

// POST /login
//
// Three possible outcomes, matching docs/adr/0008:
//   - email unknown anywhere            -> 401
//   - email known in exactly one tenant -> verify there, set cookie, 200
//   - email known in several tenants,
//     and this call has no tenant_id    -> 300 {"choose": [...]}  (no cookie yet)
//   - email known in several tenants,
//     and this call HAS a tenant_id     -> verify against that one, set cookie, 200
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ids := h.Tenants.TenantsForEmail(req.Email)
	if len(ids) == 0 {
		// Deliberately the same message whether the email is unknown or the
		// password would have been wrong — this endpoint does not leak which.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	targetID := req.TenantID
	if targetID == 0 {
		if len(ids) > 1 {
			choices := make([]tenantChoice, 0, len(ids))
			for _, id := range ids {
				t, err := h.Tenants.Get(id)
				if err != nil {
					continue
				}
				choices = append(choices, tenantChoice{t.TenantID, t.Slug, t.DisplayName})
			}
			c.JSON(http.StatusMultipleChoices, gin.H{"choose": choices})
			return
		}
		targetID = ids[0]
	}

	t, err := h.Tenants.Get(targetID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	jwtToken, err := h.loginAgainstTenant(t, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	sess, err := h.Sessions.Create(req.Email, t.TenantID, jwtToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create session"})
		return
	}
	c.SetCookie(CookieName, sess.ID, int(sessionTTL.Seconds()), "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"tenant_id": t.TenantID, "slug": t.Slug})
}

// loginAgainstTenant replays the credentials against the tenant's own
// nexwall-controller API — the only place a password is ever checked. See
// docs/adr/0008: "not a credential store."
func (h *Handler) loginAgainstTenant(t *tenant.Tenant, email, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"username": email, "password": password})
	req, err := http.NewRequest(http.MethodPost, t.InClusterWebAddr()+"/api/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errInvalidCredentials
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Token, nil
}

// GET /sso/handoff?token=...&tenant=<slug>
//
// What nexwall-partner-multitenant actually redirects a browser to (ADR
// 0009, ADR 0008's "handoff" counterpart to Login's password replay). The
// token itself is opaque here: this handler does not verify it, has no
// reason to hold the tenant's handoff secret, and forwards it verbatim to
// the target tenant's own /sso/handoff (nexwall-controller, ADR 0001
// there), which does the actual verification with the copy of the same
// secret ADR 0009 provisioned into its environment. On success, the
// tenant's JWT comes back exactly as it would from a password-replay
// login, and is wrapped in a session the same way.
func (h *Handler) Handoff(c *gin.Context) {
	token := c.Query("token")
	slug := c.Query("tenant")
	if token == "" || slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token and tenant are required"})
		return
	}

	t, err := h.Tenants.GetBySlug(slug)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	jwtToken, email, err := h.forwardHandoff(t, token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	sess, err := h.Sessions.Create(email, t.TenantID, jwtToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create session"})
		return
	}
	c.SetCookie(CookieName, sess.ID, int(sessionTTL.Seconds()), "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

// forwardHandoff calls the tenant's own /sso/handoff and returns the JWT it
// mints. email is best-effort (decoded from the JWT's own claims, not
// re-verified here -- the tenant already proved the token was good by
// minting a session off it); used only for Session.Email bookkeeping, never
// for authorization decisions.
func (h *Handler) forwardHandoff(t *tenant.Tenant, token string) (jwtToken, email string, err error) {
	req, err := http.NewRequest(http.MethodGet, t.InClusterWebAddr()+"/sso/handoff?token="+url.QueryEscape(token), nil)
	if err != nil {
		return "", "", err
	}

	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", errInvalidCredentials
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	return out.Token, jwtSubject(out.Token), nil
}

// jwtSubject reads the "id" claim out of a JWT's payload without verifying
// the signature -- safe here specifically because the token was only ever
// used for Session.Email bookkeeping, immediately after this same process
// received it directly from the tenant that just minted and vouched for it
// over the call in forwardHandoff above. Never use this pattern to make an
// authorization decision.
func jwtSubject(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	return claims.ID
}

// POST /logout
func (h *Handler) Logout(c *gin.Context) {
	if id, err := c.Cookie(CookieName); err == nil {
		h.Sessions.Delete(id)
	}
	c.SetCookie(CookieName, "", -1, "/", "", false, true)
	c.Status(http.StatusNoContent)
}

// RequireSession is middleware for the proxy route: no valid session cookie,
// no proxying.
func (h *Handler) RequireSession(c *gin.Context) {
	id, err := c.Cookie(CookieName)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
		return
	}
	sess, ok := h.Sessions.Get(id)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
		return
	}
	c.Set("session", sess)
	c.Next()
}

// ProxyToTenant is the dynamic reverse proxy from docs/adr/0008: forwards
// everything at the root path to the session's current tenant, injecting
// its JWT. No URL prefix, so nexwall-ui's asset paths need no change.
func (h *Handler) ProxyToTenant(c *gin.Context) {
	sessVal, _ := c.Get("session")
	sess := sessVal.(*Session)

	t, err := h.Tenants.Get(sess.TenantID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "tenant no longer exists"})
		return
	}

	target, err := url.Parse(t.InClusterWebAddr())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("Authorization", "Bearer "+sess.TenantJWT)
		req.Host = target.Host
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Never let a tenant's own Set-Cookie collide with our session
		// cookie's name/scope.
		resp.Header.Del("Set-Cookie")
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "tenant unreachable")
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

var errInvalidCredentials = &authError{"invalid credentials"}

type authError struct{ msg string }

func (e *authError) Error() string { return e.msg }
