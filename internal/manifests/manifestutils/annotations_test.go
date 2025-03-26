package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGCSShortLiveTokenAnnotation(t *testing.T) {
	annotations := GCSShortLiveTokenAnnotation(GCSShortLived{
		IAMServiceAccount: "test-sa",
		ProjectID:         "test-project",
	})

	assert.Equal(t, annotations["iam.gke.io/gcp-service-account"],
		"test-sa@test-project.iam.gserviceaccount.com")

}
