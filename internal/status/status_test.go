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

func TestRefreshPatchError(t *testing.T) {
	c := &statusClientStub{}
	c.PatchStatusStub = func(ctx context.Context, changed, original *v1alpha1.TempoStack) error {
		return apierrors.NewConflict(schema.GroupResource{}, original.Name,
			errors.New("patching error, likely some other thing modified this and the patch was rejected"))
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
	requeue, err := Refresh(context.Background(), c, stack, s)
	assert.True(t, requeue)
	assert.Error(t, err)
}

func TestRefreshNoError(t *testing.T) {
	c := &statusClientStub{}
	callPatchCount := 0

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
		TempoVersion:      "2.0",
		TempoQueryVersion: "main-1b50ad3",
		Conditions:        ReadyCondition(c, stack),
	}

	c.PatchStatusStub = func(ctx context.Context, changed, original *v1alpha1.TempoStack) error {
		assert.Equal(t, s, changed.Status)
		callPatchCount++
		return nil
	}

	requeue, err := Refresh(context.Background(), c, stack, &s)
	assert.False(t, requeue)
	assert.NoError(t, err)
}
