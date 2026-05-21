/*
Copyright 2023 DragonflyDB authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCopyDesiredPayload_ConfigMapDataUpdated(t *testing.T) {
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "df-liveness", Namespace: "default"},
		Data:       map[string]string{"liveness-check.sh": "echo old"},
	}
	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "df-liveness", Namespace: "default"},
		Data:       map[string]string{"liveness-check.sh": "echo new"},
	}

	copyDesiredPayload(desired, existing)

	assert.Equal(t, "echo new", existing.Data["liveness-check.sh"])
}

func TestCopyDesiredPayload_ConfigMapBinaryDataUpdated(t *testing.T) {
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "df-bin", Namespace: "default"},
		BinaryData: map[string][]byte{"k": []byte("old")},
	}
	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "df-bin", Namespace: "default"},
		BinaryData: map[string][]byte{"k": []byte("new")},
	}

	copyDesiredPayload(desired, existing)

	assert.Equal(t, []byte("new"), existing.BinaryData["k"])
}

func TestCopyDesiredPayload_ConfigMapImmutableUpdated(t *testing.T) {
	t.Run("immutable flipped", func(t *testing.T) {
		f, tr := false, true
		existing := &corev1.ConfigMap{Immutable: &f}
		desired := &corev1.ConfigMap{Immutable: &tr}

		copyDesiredPayload(desired, existing)

		assert.NotNil(t, existing.Immutable)
		assert.True(t, *existing.Immutable)
	})
}

func TestCopyDesiredPayload_StatefulSetSpecUpdated(t *testing.T) {
	one, three := int32(1), int32(3)
	existing := &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Replicas: &one}}
	desired := &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Replicas: &three}}

	copyDesiredPayload(desired, existing)

	assert.NotNil(t, existing.Spec.Replicas)
	assert.Equal(t, int32(3), *existing.Spec.Replicas)
}

func TestCopyDesiredPayload_ServiceSpecUpdated(t *testing.T) {
	existing := &corev1.Service{Spec: corev1.ServiceSpec{
		Type:  corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{{Port: 6379}},
	}}
	desired := &corev1.Service{Spec: corev1.ServiceSpec{
		Type:  corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{{Port: 6379}, {Port: 9999, Name: "admin"}},
	}}

	copyDesiredPayload(desired, existing)

	assert.Len(t, existing.Spec.Ports, 2)
	assert.Equal(t, int32(9999), existing.Spec.Ports[1].Port)
}

func TestCopyDesiredPayload_PodDisruptionBudgetSpecUpdated(t *testing.T) {
	oldMin := intstr.FromInt(1)
	newMin := intstr.FromInt(2)
	existing := &policyv1.PodDisruptionBudget{Spec: policyv1.PodDisruptionBudgetSpec{MinAvailable: &oldMin}}
	desired := &policyv1.PodDisruptionBudget{Spec: policyv1.PodDisruptionBudgetSpec{MinAvailable: &newMin}}

	copyDesiredPayload(desired, existing)

	assert.Equal(t, int32(2), existing.Spec.MinAvailable.IntVal)
}

func TestCopyDesiredPayload_NetworkPolicySpecUpdated(t *testing.T) {
	existing := &networkingv1.NetworkPolicy{Spec: networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
	}}
	desired := &networkingv1.NetworkPolicy{Spec: networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
	}}

	copyDesiredPayload(desired, existing)

	assert.Len(t, existing.Spec.PolicyTypes, 2)
}

func TestResourceSpecsEqual_ConfigMapDataDiffers(t *testing.T) {
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cm"},
		Data:       map[string]string{"k": "old"},
	}
	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cm"},
		Data:       map[string]string{"k": "new"},
	}

	assert.False(t, resourceSpecsEqual(desired, existing))
}

func TestResourceSpecsEqual_ConfigMapDataEqual(t *testing.T) {
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cm"},
		Data:       map[string]string{"k": "v"},
	}
	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cm"},
		Data:       map[string]string{"k": "v"},
	}

	assert.True(t, resourceSpecsEqual(desired, existing))
}

func TestResourceSpecsEqual_LabelsDiffer(t *testing.T) {
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cm", Labels: map[string]string{"app": "old"}},
		Data:       map[string]string{"k": "v"},
	}
	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cm", Labels: map[string]string{"app": "new"}},
		Data:       map[string]string{"k": "v"},
	}

	assert.False(t, resourceSpecsEqual(desired, existing))
}

func TestResourceSpecsEqual_PerKind(t *testing.T) {
	one, three := int32(1), int32(3)
	minA := intstr.FromInt(1)
	minB := intstr.FromInt(2)

	tests := []struct {
		name          string
		desired       client.Object
		existing      client.Object
		expectedEqual bool
	}{
		{
			name:          "StatefulSet replicas differ",
			desired:       &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Replicas: &three}},
			existing:      &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Replicas: &one}},
			expectedEqual: false,
		},
		{
			name:          "StatefulSet replicas equal",
			desired:       &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Replicas: &three}},
			existing:      &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Replicas: &three}},
			expectedEqual: true,
		},
		{
			name:          "Service ports differ",
			desired:       &corev1.Service{Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 9999}}}},
			existing:      &corev1.Service{Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 6379}}}},
			expectedEqual: false,
		},
		{
			name:          "PDB minAvailable differs",
			desired:       &policyv1.PodDisruptionBudget{Spec: policyv1.PodDisruptionBudgetSpec{MinAvailable: &minA}},
			existing:      &policyv1.PodDisruptionBudget{Spec: policyv1.PodDisruptionBudgetSpec{MinAvailable: &minB}},
			expectedEqual: false,
		},
		{
			name: "NetworkPolicy policyTypes differ",
			desired: &networkingv1.NetworkPolicy{Spec: networkingv1.NetworkPolicySpec{
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			}},
			existing: &networkingv1.NetworkPolicy{Spec: networkingv1.NetworkPolicySpec{
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			}},
			expectedEqual: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedEqual, resourceSpecsEqual(tc.desired, tc.existing))
		})
	}
}
