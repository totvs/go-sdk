package status

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SummaryOptions customizes summary derivation.
type SummaryOptions struct {
	SummaryMapping SummaryMapping
}

// SummaryOption mutates SummaryOptions.
type SummaryOption func(*SummaryOptions)

// WithSummaryMapping merges m on top of the current mapping (defaults when no prior option was given).
// Multiple WithSummaryMapping calls compose left-to-right: each one is merged over the previous result.
func WithSummaryMapping(m SummaryMapping) SummaryOption {
	return func(options *SummaryOptions) {
		options.SummaryMapping = options.SummaryMapping.Merge(m)
	}
}

// SummaryFromObject computes the normalized Summary for any controller-runtime client object.
func SummaryFromObject(obj client.Object, opts ...SummaryOption) (Summary, error) {
	if obj == nil {
		return Summary{}, fmt.Errorf("object cannot be nil")
	}

	if u, ok := obj.(*unstructured.Unstructured); ok {
		return SummaryFromUnstructured(u, opts...)
	}

	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return Summary{}, fmt.Errorf("failed to convert object to unstructured: %w", err)
	}

	u := &unstructured.Unstructured{Object: raw}
	if gvk := obj.GetObjectKind().GroupVersionKind(); gvk.Kind != "" {
		u.SetGroupVersionKind(gvk)
	}

	return SummaryFromUnstructured(u, opts...)
}

// SummaryFromUnstructured computes the normalized Summary for an unstructured object.
func SummaryFromUnstructured(u *unstructured.Unstructured, opts ...SummaryOption) (Summary, error) {
	if u == nil {
		return Summary{}, fmt.Errorf("object cannot be nil")
	}

	result, err := kstatus.Compute(u)
	if err != nil {
		return Summary{}, fmt.Errorf("failed to compute kstatus: %w", err)
	}

	conditions, err := ConditionsFromUnstructured(u)
	if err != nil {
		return Summary{}, err
	}

	observedGeneration := ObservedGenerationFromUnstructured(u)
	generation := u.GetGeneration()
	stale := generation > 0 && observedGeneration > 0 && generation != observedGeneration
	reconciling := IsConditionTrue(conditions, ConditionTypeReconciling)
	stalled := IsConditionTrue(conditions, ConditionTypeStalled)

	representative := pickRepresentativeCondition(conditions, KStatus(result.Status), stalled, reconciling)

	var reason Reason
	var message string
	if representative != nil {
		reason = Reason(representative.Reason)
		message = representative.Message
	} else {
		message = result.Message
	}

	return buildSummary(KStatus(result.Status), reason, message, stale, opts...), nil
}

// NotFoundSummary returns a Summary for a resource that was not found in the cluster.
func NotFoundSummary(reason Reason, message string, opts ...SummaryOption) Summary {
	return buildSummary(KStatusNotFound, reason, message, false, opts...)
}

func buildSummary(ks KStatus, reason Reason, message string, stale bool, opts ...SummaryOption) Summary {
	options := SummaryOptions{SummaryMapping: DefaultSummaryMapping()}
	for _, opt := range opts {
		opt(&options)
	}

	state, severity := defaultStateAndSeverity(ks)

	rule, hasRule := options.SummaryMapping[reason]
	if hasRule {
		if rule.State != "" {
			state = rule.State
		}
		if rule.Severity != "" {
			severity = rule.Severity
		}
	}

	// terminal states override mapping
	if ks == KStatusTerminating {
		state = StateTerminating
		severity = SeverityWarning
	}
	if ks == KStatusNotFound {
		state = StateNotFound
		severity = SeverityWarning
	}
	if stale && ks == KStatusInProgress {
		state = StateProgressing
		severity = SeverityInfo
	}

	return Summary{
		KStatus:  ks,
		State:    state,
		Severity: severity,
		Reason:   reason,
		Message:  message,
	}
}

func pickRepresentativeCondition(conditions []metav1.Condition, ks KStatus, stalled, reconciling bool) *metav1.Condition {
	if stalled {
		return FindCondition(conditions, ConditionTypeStalled)
	}
	if ks == KStatusTerminating {
		if c := FindCondition(conditions, ConditionTypeReady); c != nil {
			return c
		}
	}
	if reconciling {
		return FindCondition(conditions, ConditionTypeReconciling)
	}
	if c := FindCondition(conditions, ConditionTypeReady); c != nil {
		return c
	}
	for i := range conditions {
		if conditions[i].Status != metav1.ConditionTrue {
			c := conditions[i]
			return &c
		}
	}
	if len(conditions) > 0 {
		c := conditions[0]
		return &c
	}
	return nil
}

func defaultStateAndSeverity(ks KStatus) (State, Severity) {
	switch ks {
	case KStatusCurrent:
		return StateReady, SeveritySuccess
	case KStatusFailed:
		return StateError, SeverityError
	case KStatusTerminating:
		return StateTerminating, SeverityWarning
	case KStatusNotFound:
		return StateNotFound, SeverityWarning
	case KStatusInProgress:
		return StateProgressing, SeverityInfo
	default:
		return StateUnknown, SeverityWarning
	}
}
