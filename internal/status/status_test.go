package status

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
)

func TestRefreshUpdateError(t *testing.T) {
	c := &statusClientStub{}
	c.UpdateStatusStub = func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
		return apierrors.NewConflict(schema.GroupResource{}, obj.GetName(),
			errors.New("update error, likely some other thing modified this and the update was rejected"))
	}

	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-stack",
			Namespace: "some-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo: "local:2.0",
			},
		},
	}
	s := &v1alpha1.TempoStackStatus{}
	err := Refresh(context.Background(), c, stack, s)
	assert.Error(t, err)
}

func TestRefreshNoError(t *testing.T) {
	c := &statusClientStub{}
	callUpdateCount := 0

	stack := v1alpha1.TempoStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-stack",
			Namespace: "some-ns",
		},
		Spec: v1alpha1.TempoStackSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo: "local:2.0",
			},
		},
	}

	s := v1alpha1.TempoStackStatus{
		OperatorVersion: "0.1.0",
		TempoVersion:    "2.0",
		Conditions:      ReadyCondition(stack),
	}

	c.UpdateStatusStub = func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
		tempo := obj.(*v1alpha1.TempoStack)
		assert.Equal(t, s, tempo.Status)
		callUpdateCount++
		return nil
	}

	err := Refresh(context.Background(), c, stack, &s)
	assert.NoError(t, err)
}
