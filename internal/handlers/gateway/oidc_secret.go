package gateway

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
	"github.com/os-observability/tempo-operator/internal/status"
)

// GetTenantSecrets returns the list to gateway tenant secrets for a tenant mode.
// For the static mode, the secrets are fetched from externally provided secrets.
// All secrets live in the same namespace as the tempostack request.
func GetTenantSecrets(
	ctx context.Context,
	k8sClient client.Client,
	tempo *v1alpha1.TempoStack,
) ([]*manifestutils.GatewayTenantSecret, error) {
	var (
		tenantSecrets []*manifestutils.GatewayTenantSecret
		gatewaySecret corev1.Secret
	)

	for _, tenant := range tempo.Spec.Tenants.Authentication {
		key := client.ObjectKey{Name: tenant.OIDC.Secret.Name, Namespace: tempo.Namespace}
		if err := k8sClient.Get(ctx, key, &gatewaySecret); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, &status.DegradedError{
					Message: fmt.Sprintf("Missing secrets for tenant %s", tenant.TenantName),
					Reason:  v1alpha1.ReasonMissingGatewayTenantSecret,
					Requeue: true,
				}
			}
			return nil, fmt.Errorf("failed to lookup tempostack gateway tenant secret, name: %s, error: %w", key, err)
		}

		var ts *manifestutils.GatewayTenantSecret
		ts, err := extractSecret(&gatewaySecret, tenant.TenantName)
		if err != nil {
			return nil, &status.DegradedError{
				Message: "Invalid gateway tenant secret contents",
				Reason:  v1alpha1.ReasonMissingGatewayTenantSecret,
				Requeue: true,
			}
		}
		tenantSecrets = append(tenantSecrets, ts)
	}

	return tenantSecrets, nil
}

// extractSecret reads a k8s secret into a manifest tenant secret struct if valid.
func extractSecret(s *corev1.Secret, tenantName string) (*manifestutils.GatewayTenantSecret, error) {
	// Extract and validate mandatory fields
	clientID := s.Data["clientID"]
	if len(clientID) == 0 {
		return nil, fmt.Errorf("missing clientID field")
	}
	clientSecret := s.Data["clientSecret"]
	issuerCAPath := s.Data["issuerCAPath"]

	return &manifestutils.GatewayTenantSecret{
		TenantName:   tenantName,
		ClientID:     string(clientID),
		ClientSecret: string(clientSecret),
		IssuerCAPath: string(issuerCAPath),
	}, nil
}
