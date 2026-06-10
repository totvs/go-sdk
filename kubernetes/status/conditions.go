package status

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCondition builds a metav1.Condition using the SDK reason convention.
func NewCondition(conditionType string, conditionStatus metav1.ConditionStatus, reason Reason, message string, observedGeneration int64) metav1.Condition {
	return metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		ObservedGeneration: observedGeneration,
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
}

// SetCondition upserts a condition by type, following Kubernetes metav1.Condition semantics.
func SetCondition(conditions *[]metav1.Condition, condition metav1.Condition) {
	if conditions == nil {
		return
	}
	meta.SetStatusCondition(conditions, condition)
}

// RemoveCondition removes a condition by type.
func RemoveCondition(conditions *[]metav1.Condition, conditionType string) {
	if conditions == nil {
		return
	}
	meta.RemoveStatusCondition(conditions, conditionType)
}

// FindCondition returns a copy of the condition for the given type.
func FindCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if strings.EqualFold(conditions[i].Type, conditionType) {
			condition := conditions[i]
			return &condition
		}
	}
	return nil
}

// IsConditionTrue reports whether the condition type exists and has status True.
func IsConditionTrue(conditions []metav1.Condition, conditionType string) bool {
	condition := FindCondition(conditions, conditionType)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsConditionFalse reports whether the condition type exists and has status False.
func IsConditionFalse(conditions []metav1.Condition, conditionType string) bool {
	condition := FindCondition(conditions, conditionType)
	return condition != nil && condition.Status == metav1.ConditionFalse
}

// MarkReady marks a resource as reconciled and removes abnormal KStatus conditions.
func MarkReady(conditions *[]metav1.Condition, observedGeneration int64, reason Reason, message string) {
	SetCondition(conditions, NewCondition(ConditionTypeReady, metav1.ConditionTrue, reason, message, observedGeneration))
	RemoveCondition(conditions, ConditionTypeReconciling)
	RemoveCondition(conditions, ConditionTypeStalled)
}

// MarkReconciling marks a resource as actively reconciling.
func MarkReconciling(conditions *[]metav1.Condition, observedGeneration int64, reason Reason, message string) {
	SetCondition(conditions, NewCondition(ConditionTypeReady, metav1.ConditionFalse, reason, message, observedGeneration))
	SetCondition(conditions, NewCondition(ConditionTypeReconciling, metav1.ConditionTrue, reason, message, observedGeneration))
	RemoveCondition(conditions, ConditionTypeStalled)
}

// MarkWaiting marks a resource as not ready because it is waiting for an external input or dependency.
func MarkWaiting(conditions *[]metav1.Condition, observedGeneration int64, reason Reason, message string) {
	SetCondition(conditions, NewCondition(ConditionTypeReady, metav1.ConditionFalse, reason, message, observedGeneration))
	RemoveCondition(conditions, ConditionTypeReconciling)
	RemoveCondition(conditions, ConditionTypeStalled)
}

// MarkStalled marks a resource as blocked or failed.
func MarkStalled(conditions *[]metav1.Condition, observedGeneration int64, reason Reason, message string) {
	SetCondition(conditions, NewCondition(ConditionTypeReady, metav1.ConditionFalse, reason, message, observedGeneration))
	SetCondition(conditions, NewCondition(ConditionTypeStalled, metav1.ConditionTrue, reason, message, observedGeneration))
	RemoveCondition(conditions, ConditionTypeReconciling)
}

// MarkTerminating marks the resource conditions as terminating. KStatus itself is
// still derived from metadata.deletionTimestamp when a full object is evaluated.
func MarkTerminating(conditions *[]metav1.Condition, observedGeneration int64, message string) {
	SetCondition(conditions, NewCondition(ConditionTypeReady, metav1.ConditionFalse, Reasons.Terminating, message, observedGeneration))
	SetCondition(conditions, NewCondition(ConditionTypeReconciling, metav1.ConditionTrue, Reasons.Terminating, message, observedGeneration))
	RemoveCondition(conditions, ConditionTypeStalled)
}
