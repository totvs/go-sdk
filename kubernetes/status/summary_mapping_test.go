package status

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ── Default Mark* → DefaultSummaryMapping inference ──────────────────────────

func TestDefaultMapping_MarkReadyProducesCurrentAndReady(t *testing.T) {
	// Arrange
	conditions := []metav1.Condition{}
	MarkReady(&conditions, 1, Reasons.Reconciled, "reconciled")
	obj := newStatusObject(1, 1, conditions...)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusCurrent, summary.KStatus)
	require.Equal(t, StateReady, summary.State)
	require.Equal(t, SeveritySuccess, summary.Severity)
	require.Equal(t, Reasons.Reconciled, summary.Reason)
}

func TestDefaultMapping_MarkReconcilingProducesInProgressAndProgressing(t *testing.T) {
	// Arrange
	conditions := []metav1.Condition{}
	MarkReconciling(&conditions, 1, Reasons.Reconciling, "installing")
	obj := newStatusObject(1, 1, conditions...)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusInProgress, summary.KStatus)
	require.Equal(t, StateProgressing, summary.State)
	require.Equal(t, SeverityInfo, summary.Severity)
	require.Equal(t, Reasons.Reconciling, summary.Reason)
}

func TestDefaultMapping_MarkStalledProducesFailedAndError(t *testing.T) {
	// Arrange — reason not in any mapping; KStatus default for Failed applies
	conditions := []metav1.Condition{}
	MarkStalled(&conditions, 1, "SomeDomainReason", "unrecognized failure")
	obj := newStatusObject(1, 1, conditions...)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert — no mapping match → KStatusFailed default: state=error, severity=error
	require.NoError(t, err)
	require.Equal(t, KStatusFailed, summary.KStatus)
	require.Equal(t, StateError, summary.State)
	require.Equal(t, SeverityError, summary.Severity)
}

func TestDefaultMapping_MarkWaitingProducesInProgress(t *testing.T) {
	// Arrange — MarkWaiting with unknown reason — no mapping match, falls back to KStatus default
	conditions := []metav1.Condition{}
	MarkWaiting(&conditions, 1, Reasons.Unknown, "waiting")
	obj := newStatusObject(1, 1, conditions...)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusInProgress, summary.KStatus)
	require.Equal(t, StateUnknown, summary.State) // Unknown reason overrides KStatus default
	require.Equal(t, SeverityWarning, summary.Severity)
}

// ── Common Reasons → commonSummaryMapping inference ───────────────────────────

func TestCommonMapping_DependencyNotFoundInferredFromCondition(t *testing.T) {
	// Arrange — operator marks stalled with DependencyNotFound (e.g. driver CRD missing)
	conditions := []metav1.Condition{}
	MarkStalled(&conditions, 1, Reasons.DependencyNotFound, "driver CRD not found")
	obj := newStatusObject(1, 1, conditions...)

	// Act — no custom mapping — commonSummaryMapping covers DependencyNotFound
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusFailed, summary.KStatus)
	require.Equal(t, StateError, summary.State)
	require.Equal(t, SeverityError, summary.Severity)
	require.Equal(t, Reasons.DependencyNotFound, summary.Reason)
}

func TestCommonMapping_DependencyUnavailableInferredFromCondition(t *testing.T) {
	// Arrange — dependency exists but is not ready
	conditions := []metav1.Condition{}
	MarkStalled(&conditions, 1, Reasons.DependencyUnavailable, "database not ready")
	obj := newStatusObject(1, 1, conditions...)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusFailed, summary.KStatus)
	require.Equal(t, StateError, summary.State)
	require.Equal(t, SeverityWarning, summary.Severity) // warning, not error — dep exists, just unavailable
	require.Equal(t, Reasons.DependencyUnavailable, summary.Reason)
}

func TestCommonMapping_InvalidConfigurationInferredFromCondition(t *testing.T) {
	// Arrange
	conditions := []metav1.Condition{}
	MarkStalled(&conditions, 1, Reasons.InvalidConfiguration, "invalid helm values")
	obj := newStatusObject(1, 1, conditions...)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, StateError, summary.State)
	require.Equal(t, SeverityError, summary.Severity)
	require.Equal(t, Reasons.InvalidConfiguration, summary.Reason)
}

func TestCommonMapping_PreconditionNotMetInferredFromCondition(t *testing.T) {
	// Arrange — resource waiting for a precondition (e.g. cert not yet issued)
	conditions := []metav1.Condition{}
	MarkWaiting(&conditions, 1, Reasons.PreconditionNotMet, "TLS cert not yet issued")
	obj := newStatusObject(1, 1, conditions...)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusInProgress, summary.KStatus)
	require.Equal(t, StateWaiting, summary.State)
	require.Equal(t, SeverityWarning, summary.Severity)
	require.Equal(t, Reasons.PreconditionNotMet, summary.Reason)
}

func TestCommonMapping_PermissionDeniedInferredFromCondition(t *testing.T) {
	// Arrange
	conditions := []metav1.Condition{}
	MarkStalled(&conditions, 1, Reasons.PermissionDenied, "service account missing RBAC")
	obj := newStatusObject(1, 1, conditions...)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, StateError, summary.State)
	require.Equal(t, SeverityError, summary.Severity)
	require.Equal(t, Reasons.PermissionDenied, summary.Reason)
}

func TestCommonMapping_TimeoutInferredFromCondition(t *testing.T) {
	// Arrange
	conditions := []metav1.Condition{}
	MarkStalled(&conditions, 1, Reasons.Timeout, "webhook took too long")
	obj := newStatusObject(1, 1, conditions...)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, StateError, summary.State)
	require.Equal(t, SeverityWarning, summary.Severity)
	require.Equal(t, Reasons.Timeout, summary.Reason)
}

func TestCommonMapping_ConflictInferredFromCondition(t *testing.T) {
	// Arrange
	conditions := []metav1.Condition{}
	MarkStalled(&conditions, 1, Reasons.Conflict, "resource locked by another controller")
	obj := newStatusObject(1, 1, conditions...)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, StateError, summary.State)
	require.Equal(t, SeverityWarning, summary.Severity)
	require.Equal(t, Reasons.Conflict, summary.Reason)
}
