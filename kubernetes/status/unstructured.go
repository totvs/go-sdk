package status

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ConditionsFromUnstructured extracts status.conditions as metav1.Condition.
func ConditionsFromUnstructured(u *unstructured.Unstructured) ([]metav1.Condition, error) {
	if u == nil {
		return nil, fmt.Errorf("object cannot be nil")
	}

	rawConditions, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if err != nil {
		return nil, fmt.Errorf("failed to read status.conditions: %w", err)
	}
	if !found {
		return nil, nil
	}

	conditions := make([]metav1.Condition, 0, len(rawConditions))
	for i, rawCondition := range rawConditions {
		rawConditionMap, ok := rawCondition.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("status.conditions[%d] is not an object", i)
		}

		var condition metav1.Condition
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(rawConditionMap, &condition); err != nil {
			return nil, fmt.Errorf("failed to convert status.conditions[%d]: %w", i, err)
		}
		conditions = append(conditions, condition)
	}

	return conditions, nil
}

// ObservedGenerationFromUnstructured extracts status.observedGeneration.
func ObservedGenerationFromUnstructured(u *unstructured.Unstructured) int64 {
	if u == nil {
		return 0
	}

	observedGeneration, found, err := unstructured.NestedInt64(u.Object, "status", "observedGeneration")
	if err == nil && found {
		return observedGeneration
	}

	observedGenerationFloat, found, err := unstructured.NestedFloat64(u.Object, "status", "observedGeneration")
	if err == nil && found {
		return int64(observedGenerationFloat)
	}

	return 0
}
