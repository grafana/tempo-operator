package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGCSShortLiveTokenAnnotation(t *testing.T) {
	annotations := GCSShortLiveTokenAnnotation(GCS{
		IAMServiceAccount: "test-sa",
		ProjectID:         "test-project",
	})

	assert.Equal(t, annotations["iam.gke.io/gcp-service-account"],
		"test-sa@test-project.iam.gserviceaccount.com")

}

func TestAzureShortLiveTokenAnnotation(t *testing.T) {
	annotations := AzureShortLiveTokenAnnotation(AzureStorage{
		TenantID: "test-tenant",
		ClientID: "test-client",
	})

	assert.Equal(t, "test-client", annotations["azure.workload.identity/client-id"])
	assert.Equal(t, "test-tenant", annotations["azure.workload.identity/tenant-id"])
}
