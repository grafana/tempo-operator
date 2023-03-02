package status

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
)

func TestRefreshTagError(t *testing.T) {
	c := &statusClientStub{}
	stack := v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-stack",
			Namespace: "some-ns",
		},
		Spec: v1alpha1.MicroservicesSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo: "",
			},
		},
	}
	s := &v1alpha1.MicroservicesStatus{}
	requeue, err := Refresh(context.Background(), c, stack, s)
	assert.False(t, requeue)
	assert.Error(t, err)
}

func TestRefreshPatchError(t *testing.T) {
	c := &statusClientStub{}
	c.PatchStatusStub = func(ctx context.Context, changed, original *v1alpha1.Microservices) error {
		return apierrors.NewConflict(schema.GroupResource{}, original.Name,
			errors.New("patching error, likely some other thing modified this and the patch was rejected"))
	}

	stack := v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-stack",
			Namespace: "some-ns",
		},
		Spec: v1alpha1.MicroservicesSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo: "local:2.0",
			},
		},
	}
	s := &v1alpha1.MicroservicesStatus{}
	requeue, err := Refresh(context.Background(), c, stack, s)
	assert.True(t, requeue)
	assert.Error(t, err)
}

func TestRefreshNoError(t *testing.T) {
	c := &statusClientStub{}
	callPatchCount := 0

	stack := v1alpha1.Microservices{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-stack",
			Namespace: "some-ns",
		},
		Spec: v1alpha1.MicroservicesSpec{
			Images: configv1alpha1.ImagesSpec{
				Tempo: "local:2.0",
			},
		},
	}

	s := v1alpha1.MicroservicesStatus{
		TempoVersion: "2.0",
		Conditions:   ReadyCondition(c, stack),
	}

	c.PatchStatusStub = func(ctx context.Context, changed, original *v1alpha1.Microservices) error {
		assert.Equal(t, changed.Status, s)
		callPatchCount++
		return nil
	}

	requeue, err := Refresh(context.Background(), c, stack, &s)
	assert.False(t, requeue)
	assert.NoError(t, err)
}
