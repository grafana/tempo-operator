package grafana

import (
	"encoding/json"
	"fmt"
	"testing"

	grafanav1 "github.com/grafana-operator/grafana-operator/v5/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

func TestBuildGrafanaDatasource(t *testing.T) {
	datasource, err := BuildGrafanaDatasource(manifestutils.Params{Tempo: v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "tempo",
		},
		Spec: v1alpha1.TempoStackSpec{},
	}})
	labels := manifestutils.CommonLabels("test")

	require.NoError(t, err)

	assert.NotNil(t, datasource)
	assert.Equal(t, &grafanav1.GrafanaDatasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "tempo",
			Labels:    labels,
		},
		Spec: grafanav1.GrafanaDatasourceSpec{
			Datasource: &grafanav1.GrafanaDatasourceInternal{
				Access:   "proxy",
				Name:     "test",
				Type:     "tempo",
				URL:      "http://tempo-test-query-frontend.tempo.svc.cluster.local:3200",
				JSONData: json.RawMessage(fmt.Sprintf(`{"tlsSkipVerify": %t}`, true)),
			},
			InstanceSelector: &metav1.LabelSelector{},
		},
	}, datasource)
}
