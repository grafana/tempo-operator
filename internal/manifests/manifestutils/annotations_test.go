package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAzureShortLiveTokenAnnotation(t *testing.T) {
	annotations := AzureShortLiveTokenAnnotation(AzureStorage{
		TenantID: "test-tenant",
		ClientID: "test-client",
	})

	assert.Equal(t, "test-client", annotations["azure.workload.identity/client-id"])
	assert.Equal(t, "test-tenant", annotations["azure.workload.identity/tenant-id"])
}
