package status

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Compute returns the KStatus for any controller-runtime client object.
func Compute(obj client.Object) (KStatus, error) {
	if obj == nil {
		return KStatusUnknown, fmt.Errorf("object cannot be nil")
	}
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return ComputeFromUnstructured(u)
	}
	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return KStatusUnknown, fmt.Errorf("failed to convert object to unstructured: %w", err)
	}
	u := &unstructured.Unstructured{Object: raw}
	if gvk := obj.GetObjectKind().GroupVersionKind(); gvk.Kind != "" {
		u.SetGroupVersionKind(gvk)
	}
	return ComputeFromUnstructured(u)
}

// ComputeFromUnstructured returns the KStatus for an unstructured object.
func ComputeFromUnstructured(u *unstructured.Unstructured) (KStatus, error) {
	if u == nil {
		return KStatusUnknown, fmt.Errorf("object cannot be nil")
	}
	result, err := kstatus.Compute(u)
	if err != nil {
		return KStatusUnknown, fmt.Errorf("failed to compute kstatus: %w", err)
	}
	return KStatus(result.Status), nil
}
