package controller

import (
	"context"
	"testing"

	dfv1alpha1 "github.com/dragonflydb/dragonfly-operator/api/v1alpha1"
	"github.com/dragonflydb/dragonfly-operator/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func newTestDFI(t *testing.T, df *dfv1alpha1.Dragonfly, objs ...client.Object) *DragonflyInstance {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, dfv1alpha1.AddToScheme(scheme))

	allObjs := []client.Object{df}
	allObjs = append(allObjs, objs...)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(allObjs...).
		Build()

	return &DragonflyInstance{
		df:     df,
		client: c,
		log:    zap.New(zap.UseDevMode(true)),
		scheme: scheme,
	}
}

func makeDragonfly(name, ns string, probeRefs ...*corev1.LocalObjectReference) *dfv1alpha1.Dragonfly {
	df := &dfv1alpha1.Dragonfly{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       dfv1alpha1.DragonflySpec{Replicas: 1},
	}
	if len(probeRefs) > 0 && probeRefs[0] != nil {
		df.Spec.CustomLivenessProbeConfigMap = probeRefs[0]
	}
	if len(probeRefs) > 1 && probeRefs[1] != nil {
		df.Spec.CustomReadinessProbeConfigMap = probeRefs[1]
	}
	if len(probeRefs) > 2 && probeRefs[2] != nil {
		df.Spec.CustomStartupProbeConfigMap = probeRefs[2]
	}
	return df
}

func makeConfigMap(name, ns string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Data:       map[string]string{"check.sh": "#!/bin/sh\nexit 0"},
	}
}

func TestCustomProbeConfigMapNames(t *testing.T) {
	df := makeDragonfly("test", "default",
		&corev1.LocalObjectReference{Name: "my-liveness"},
		nil,
		&corev1.LocalObjectReference{Name: "my-startup"},
	)
	dfi := newTestDFI(t, df)
	names := dfi.customProbeConfigMapNames()
	assert.Equal(t, []string{"my-liveness", "my-startup"}, names)
}

func TestCustomProbeConfigMapNames_Empty(t *testing.T) {
	df := makeDragonfly("test", "default")
	dfi := newTestDFI(t, df)
	names := dfi.customProbeConfigMapNames()
	assert.Empty(t, names)
}

func TestEnsureFinalizerOnConfigMap(t *testing.T) {
	cm := makeConfigMap("my-probe", "default")
	df := makeDragonfly("test", "default",
		&corev1.LocalObjectReference{Name: "my-probe"},
	)
	dfi := newTestDFI(t, df, cm)
	ctx := context.Background()

	require.NoError(t, dfi.ensureFinalizerOnConfigMap(ctx, "my-probe"))

	var updated corev1.ConfigMap
	require.NoError(t, dfi.client.Get(ctx, client.ObjectKeyFromObject(cm), &updated))
	assert.True(t, controllerutil.ContainsFinalizer(&updated, resources.ProbeConfigMapFinalizer))
}

func TestEnsureFinalizerOnConfigMap_AlreadyPresent(t *testing.T) {
	cm := makeConfigMap("my-probe", "default")
	controllerutil.AddFinalizer(cm, resources.ProbeConfigMapFinalizer)
	df := makeDragonfly("test", "default")
	dfi := newTestDFI(t, df, cm)
	ctx := context.Background()

	require.NoError(t, dfi.ensureFinalizerOnConfigMap(ctx, "my-probe"))

	var updated corev1.ConfigMap
	require.NoError(t, dfi.client.Get(ctx, client.ObjectKeyFromObject(cm), &updated))
	assert.True(t, controllerutil.ContainsFinalizer(&updated, resources.ProbeConfigMapFinalizer))
}

func TestEnsureFinalizerOnConfigMap_NotFound(t *testing.T) {
	df := makeDragonfly("test", "default")
	dfi := newTestDFI(t, df)
	ctx := context.Background()

	// Should not error on missing ConfigMap
	require.NoError(t, dfi.ensureFinalizerOnConfigMap(ctx, "nonexistent"))
}

func TestRemoveFinalizerFromConfigMap(t *testing.T) {
	cm := makeConfigMap("my-probe", "default")
	controllerutil.AddFinalizer(cm, resources.ProbeConfigMapFinalizer)
	df := makeDragonfly("test", "default")
	dfi := newTestDFI(t, df, cm)
	ctx := context.Background()

	require.NoError(t, dfi.removeFinalizerFromConfigMap(ctx, "my-probe"))

	var updated corev1.ConfigMap
	require.NoError(t, dfi.client.Get(ctx, client.ObjectKeyFromObject(cm), &updated))
	assert.False(t, controllerutil.ContainsFinalizer(&updated, resources.ProbeConfigMapFinalizer))
}

func TestRemoveFinalizerFromConfigMap_OtherCRReferences(t *testing.T) {
	cm := makeConfigMap("shared-probe", "default")
	controllerutil.AddFinalizer(cm, resources.ProbeConfigMapFinalizer)

	df1 := makeDragonfly("df1", "default")
	df2 := makeDragonfly("df2", "default",
		&corev1.LocalObjectReference{Name: "shared-probe"},
	)
	dfi := newTestDFI(t, df1, cm, df2)
	ctx := context.Background()

	// Should NOT remove finalizer because df2 still references it
	require.NoError(t, dfi.removeFinalizerFromConfigMap(ctx, "shared-probe"))

	var updated corev1.ConfigMap
	require.NoError(t, dfi.client.Get(ctx, client.ObjectKeyFromObject(cm), &updated))
	assert.True(t, controllerutil.ContainsFinalizer(&updated, resources.ProbeConfigMapFinalizer),
		"finalizer should remain because another CR references the ConfigMap")
}

func TestReconcileProbeConfigMapFinalizers_AddsAndRemoves(t *testing.T) {
	cmReferenced := makeConfigMap("referenced", "default")
	cmOrphaned := makeConfigMap("orphaned", "default")
	controllerutil.AddFinalizer(cmOrphaned, resources.ProbeConfigMapFinalizer)

	df := makeDragonfly("test", "default",
		&corev1.LocalObjectReference{Name: "referenced"},
	)
	dfi := newTestDFI(t, df, cmReferenced, cmOrphaned)
	ctx := context.Background()

	require.NoError(t, dfi.reconcileProbeConfigMapFinalizers(ctx))

	// Referenced ConfigMap should have finalizer
	var updated corev1.ConfigMap
	require.NoError(t, dfi.client.Get(ctx, client.ObjectKeyFromObject(cmReferenced), &updated))
	assert.True(t, controllerutil.ContainsFinalizer(&updated, resources.ProbeConfigMapFinalizer))

	// Orphaned ConfigMap should have finalizer removed
	require.NoError(t, dfi.client.Get(ctx, client.ObjectKeyFromObject(cmOrphaned), &updated))
	assert.False(t, controllerutil.ContainsFinalizer(&updated, resources.ProbeConfigMapFinalizer))
}
