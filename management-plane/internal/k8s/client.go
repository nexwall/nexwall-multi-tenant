// Package k8s wraps client-go + the Helm SDK so the rest of the Management
// Plane never imports them directly. Phase 2 work item — see
// docs/phases/phase-2-management-plane-mvp.md.
package k8s

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Client struct {
	clientset *kubernetes.Clientset
}

// NewInClusterClient assumes the Management Plane runs as a Pod in the
// cluster it manages, using its own ServiceAccount (RBAC scoped to
// Namespaces/Deployments/Secrets/PVCs only — see infra/k3s/README.md for
// the ClusterRole this needs).
func NewInClusterClient() (*Client, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{clientset: cs}, nil
}

// CreateTenantNamespace creates the Namespace + ResourceQuota + NetworkPolicy
// for a new tenant. Stub — implement in Phase 2 against the real quota
// numbers once one pilot tenant's actual usage (Phase 1) gives us a baseline.
func (c *Client) CreateTenantNamespace(ctx context.Context, namespace string) error {
	panic("not implemented — see docs/phases/phase-2-management-plane-mvp.md")
}

// InstallOrUpgradeRelease shells out to the Helm SDK (helm.sh/helm/v3/pkg/action)
// to install/upgrade charts/nexwall-controller with tenant-specific values.
// Stub — implement alongside CreateTenantNamespace.
func (c *Client) InstallOrUpgradeRelease(ctx context.Context, namespace, releaseName string, values map[string]interface{}) error {
	panic("not implemented — see docs/phases/phase-2-management-plane-mvp.md")
}

// DeploymentStatus is a minimal shape for GET /tenants/:id/status.
type DeploymentStatus struct {
	Name     string
	Ready    bool
	Restarts int
}

func (c *Client) GetTenantStatus(ctx context.Context, namespace string) ([]DeploymentStatus, error) {
	panic("not implemented — see docs/phases/phase-2-management-plane-mvp.md")
}
