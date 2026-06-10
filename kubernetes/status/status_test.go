package status

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ── Write Helpers ────────────────────────────────────────────────────────────

func TestMarkReady_RemovesAbnormalConditionsAndSetsReady(t *testing.T) {
	// Arrange
	conditions := []metav1.Condition{}
	MarkReconciling(&conditions, 1, Reasons.Reconciling, "reconciling")
	MarkStalled(&conditions, 1, "ArgumentsInvalid", "arguments invalid")

	// Act
	MarkReady(&conditions, 2, Reasons.Reconciled, "ready")

	// Assert
	require.Len(t, conditions, 1)
	require.True(t, IsConditionTrue(conditions, ConditionTypeReady))
	require.Nil(t, FindCondition(conditions, ConditionTypeReconciling))
	require.Nil(t, FindCondition(conditions, ConditionTypeStalled))

	ready := FindCondition(conditions, ConditionTypeReady)
	require.Equal(t, string(Reasons.Reconciled), ready.Reason)
	require.Equal(t, int64(2), ready.ObservedGeneration)
}

func TestMarkReconciling_SetsReconcilingTypeAndRemovesStalled(t *testing.T) {
	// Arrange
	conditions := []metav1.Condition{}
	MarkStalled(&conditions, 1, "ArgumentsInvalid", "arguments invalid")

	// Act
	MarkReconciling(&conditions, 2, Reasons.Reconciling, "installing")

	// Assert
	require.True(t, IsConditionFalse(conditions, ConditionTypeReady))
	require.True(t, IsConditionTrue(conditions, ConditionTypeReconciling))
	require.Nil(t, FindCondition(conditions, ConditionTypeStalled))

	reconciling := FindCondition(conditions, ConditionTypeReconciling)
	require.Equal(t, string(Reasons.Reconciling), reconciling.Reason)
	require.Equal(t, int64(2), reconciling.ObservedGeneration)
}

func TestMarkStalled_SetsStalledTypeAndRemovesReconciling(t *testing.T) {
	// Arrange
	conditions := []metav1.Condition{}
	MarkReconciling(&conditions, 1, Reasons.Reconciling, "installing")

	// Act
	MarkStalled(&conditions, 2, "ArgumentsInvalid", "arguments invalid")

	// Assert
	require.True(t, IsConditionFalse(conditions, ConditionTypeReady))
	require.True(t, IsConditionTrue(conditions, ConditionTypeStalled))
	require.Nil(t, FindCondition(conditions, ConditionTypeReconciling))

	stalled := FindCondition(conditions, ConditionTypeStalled)
	require.Equal(t, "ArgumentsInvalid", stalled.Reason)
	require.Equal(t, int64(2), stalled.ObservedGeneration)
}

func TestMarkWaiting_SetsReadyFalseAndRemovesAbnormalConditions(t *testing.T) {
	// Arrange
	conditions := []metav1.Condition{}
	MarkReconciling(&conditions, 1, Reasons.Reconciling, "installing")

	// Act
	MarkWaiting(&conditions, 2, "PendingApproval", "awaiting approval")

	// Assert
	require.True(t, IsConditionFalse(conditions, ConditionTypeReady))
	require.Nil(t, FindCondition(conditions, ConditionTypeReconciling))
	require.Nil(t, FindCondition(conditions, ConditionTypeStalled))

	ready := FindCondition(conditions, ConditionTypeReady)
	require.Equal(t, "PendingApproval", ready.Reason)
	require.Equal(t, int64(2), ready.ObservedGeneration)
}

func TestMarkTerminating_SetsReconcilingAndReadyFalse(t *testing.T) {
	// Arrange
	conditions := []metav1.Condition{}

	// Act
	MarkTerminating(&conditions, 1, "terminating")

	// Assert
	require.True(t, IsConditionFalse(conditions, ConditionTypeReady))
	require.True(t, IsConditionTrue(conditions, ConditionTypeReconciling))
	require.Nil(t, FindCondition(conditions, ConditionTypeStalled))
}

// ── Domain Conditions ────────────────────────────────────────────────────────

func TestSetCondition_DomainConditionCoexistsWithKStatusConditions(t *testing.T) {
	// Arrange
	const conditionApproved = "Approved"
	conditions := []metav1.Condition{}
	MarkWaiting(&conditions, 1, "PendingApproval", "awaiting approval")

	// Act
	SetCondition(&conditions, NewCondition(
		conditionApproved,
		metav1.ConditionFalse,
		"PendingApproval",
		"waiting for approval",
		1,
	))

	// Assert — domain condition exists alongside KStatus conditions
	require.Len(t, conditions, 2)
	require.True(t, IsConditionFalse(conditions, conditionApproved))
	require.True(t, IsConditionFalse(conditions, ConditionTypeReady))
}

func TestSetCondition_DomainConditionIsReadByType(t *testing.T) {
	// Arrange
	const conditionJoined = "Joined"
	conditions := []metav1.Condition{}

	// Act
	domainCondition := NewCondition(conditionJoined, metav1.ConditionTrue, Reasons.Reconciled, "cluster joined", 1)
	SetCondition(&conditions, domainCondition)

	// Assert — read using the condition's own type, no string literal
	found := FindCondition(conditions, domainCondition.Type)
	require.NotNil(t, found)
	require.True(t, IsConditionTrue(conditions, domainCondition.Type))
}

// ── Summary — Ready States ───────────────────────────────────────────────────

func TestSummaryFromUnstructured_CurrentWhenReadyAndObservedGenerationMatch(t *testing.T) {
	// Arrange
	obj := newStatusObject(2, 2,
		condition(ConditionTypeReady, metav1.ConditionTrue, Reasons.Reconciled, "ready", 2),
	)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusCurrent, summary.KStatus)
	require.Equal(t, StateReady, summary.State)
	require.Equal(t, SeveritySuccess, summary.Severity)
	require.Equal(t, Reasons.Reconciled, summary.Reason)
}

func TestSummaryFromUnstructured_InProgressWhenReadyButGenerationIsStale(t *testing.T) {
	// Arrange — Ready=True but generation advanced, controller hasn't caught up
	obj := newStatusObject(4, 3,
		condition(ConditionTypeReady, metav1.ConditionTrue, Reasons.Reconciled, "previously ready", 3),
	)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert — stale Ready=True must not be treated as Current
	require.NoError(t, err)
	require.Equal(t, KStatusInProgress, summary.KStatus)
	require.Equal(t, StateProgressing, summary.State)
	require.Equal(t, SeverityInfo, summary.Severity)
}

// ── Summary — Progressing States ─────────────────────────────────────────────

func TestSummaryFromUnstructured_InProgressWhenReconcilingTypeIsTrue(t *testing.T) {
	// Arrange — Reconciling as type (not as reason on Ready)
	obj := newStatusObject(1, 1,
		condition(ConditionTypeReady, metav1.ConditionFalse, Reasons.Reconciling, "installing", 1),
		condition(ConditionTypeReconciling, metav1.ConditionTrue, Reasons.Reconciling, "installing", 1),
	)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusInProgress, summary.KStatus)
	require.Equal(t, StateProgressing, summary.State)
	require.Equal(t, SeverityInfo, summary.Severity)
	require.Equal(t, Reasons.Reconciling, summary.Reason)
}

func TestSummaryFromUnstructured_InProgressWhenNotDeployedYet(t *testing.T) {
	// Arrange
	obj := newStatusObject(1, 1,
		condition(ConditionTypeReady, metav1.ConditionFalse, "NotDeployed", "not deployed", 1),
	)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusInProgress, summary.KStatus)
	require.Equal(t, StateProgressing, summary.State)
	require.Equal(t, SeverityInfo, summary.Severity)
}

// ── Summary — Waiting States ─────────────────────────────────────────────────

func TestSummaryFromUnstructured_WaitingWhenPendingApproval(t *testing.T) {
	// Arrange — PendingApproval is a domain reason, injected via custom mapping
	const reasonPendingApproval Reason = "PendingApproval"
	mapping := SummaryMapping{
		reasonPendingApproval: {State: StateWaiting, Severity: SeverityWarning},
	}
	obj := newStatusObject(1, 1,
		condition(ConditionTypeReady, metav1.ConditionFalse, reasonPendingApproval, "waiting for approval", 1),
	)

	// Act
	summary, err := SummaryFromUnstructured(obj, WithSummaryMapping(mapping))

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusInProgress, summary.KStatus)
	require.Equal(t, StateWaiting, summary.State)
	require.Equal(t, SeverityWarning, summary.Severity)
	require.Equal(t, reasonPendingApproval, summary.Reason)
}

func TestSummaryFromUnstructured_WaitingWhenCustomReasonRegisteredInMapping(t *testing.T) {
	// Arrange — domain-specific waiting reason injected via mapping
	const customReason Reason = "PendingJoin"
	mapping := SummaryMapping{
		customReason: {State: StateWaiting, Severity: SeverityWarning},
	}
	obj := newStatusObject(1, 1,
		condition(ConditionTypeReady, metav1.ConditionFalse, customReason, "waiting for join", 1),
	)

	// Act
	summary, err := SummaryFromUnstructured(obj, WithSummaryMapping(mapping))

	// Assert
	require.NoError(t, err)
	require.Equal(t, StateWaiting, summary.State)
	require.Equal(t, SeverityWarning, summary.Severity)
}

// ── Summary — Error States ───────────────────────────────────────────────────

func TestSummaryFromUnstructured_FailedWhenStalledTypeIsTrue(t *testing.T) {
	// Arrange — Stalled=True signals terminal failure to KStatus
	obj := newStatusObject(1, 1,
		condition(ConditionTypeReady, metav1.ConditionFalse, "ArgumentsInvalid", "arguments invalid", 1),
		condition(ConditionTypeStalled, metav1.ConditionTrue, "ArgumentsInvalid", "arguments invalid", 1),
	)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusFailed, summary.KStatus)
	require.Equal(t, StateError, summary.State)
	require.Equal(t, SeverityError, summary.Severity)
	require.Equal(t, Reason("ArgumentsInvalid"), summary.Reason)
}

func TestSummaryFromUnstructured_ErrorWhenRequirementsUnsatisfied(t *testing.T) {
	// Arrange
	obj := newStatusObject(1, 1,
		condition(ConditionTypeReady, metav1.ConditionFalse, "RequirementsUnsatisfied", "requirements not met", 1),
		condition(ConditionTypeStalled, metav1.ConditionTrue, "RequirementsUnsatisfied", "requirements not met", 1),
	)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, KStatusFailed, summary.KStatus)
	require.Equal(t, StateError, summary.State)
	require.Equal(t, SeverityError, summary.Severity)
}

func TestSummaryFromUnstructured_ErrorWhenArgumentsInvalid(t *testing.T) {
	// Arrange
	obj := newStatusObject(1, 1,
		condition(ConditionTypeReady, metav1.ConditionFalse, "ArgumentsInvalid", "invalid arguments", 1),
		condition(ConditionTypeStalled, metav1.ConditionTrue, "ArgumentsInvalid", "invalid arguments", 1),
	)

	// Act
	summary, err := SummaryFromUnstructured(obj)

	// Assert
	require.NoError(t, err)
	require.Equal(t, StateError, summary.State)
	require.Equal(t, SeverityError, summary.Severity)
}

// ── Summary — Not Found ──────────────────────────────────────────────────────

func TestNotFoundSummary_ReturnsNotFoundState(t *testing.T) {
	// Arrange — resource does not exist in cluster

	// Act
	summary := NotFoundSummary(Reasons.Unknown, "resource was not found")

	// Assert
	require.Equal(t, KStatusNotFound, summary.KStatus)
	require.Equal(t, StateNotFound, summary.State)
	require.Equal(t, SeverityWarning, summary.Severity)
	require.Equal(t, Reasons.Unknown, summary.Reason)
	require.Equal(t, "resource was not found", summary.Message)
}

// ── Summary — Custom Mapping ─────────────────────────────────────────────────

func TestSummaryFromUnstructured_CustomMappingOverridesDefaultSeverity(t *testing.T) {
	// Arrange — domain wants OwnerApprovalRequired as error severity
	const customReason Reason = "OwnerApprovalRequired"
	customMapping := SummaryMapping{
		customReason: {State: StateWaiting, Severity: SeverityError},
	}
	obj := newStatusObject(1, 1,
		condition(ConditionTypeReady, metav1.ConditionFalse, customReason, "owner must approve", 1),
	)

	// Act
	summary, err := SummaryFromUnstructured(obj, WithSummaryMapping(customMapping))

	// Assert
	require.NoError(t, err)
	require.Equal(t, StateWaiting, summary.State)
	require.Equal(t, SeverityError, summary.Severity) // overridden
}

func TestSummaryFromUnstructured_CustomMappingDoesNotAffectDefaultReasons(t *testing.T) {
	// Arrange — custom mapping adds new reason, default reasons unchanged
	const customReason Reason = "ExternalDependencyMissing"
	customMapping := SummaryMapping{
		customReason: {State: StateError, Severity: SeverityError},
	}
	const reasonPendingApproval Reason = "PendingApproval"
	defaultMapping := SummaryMapping{
		reasonPendingApproval: {State: StateWaiting, Severity: SeverityWarning},
	}
	obj := newStatusObject(1, 1,
		condition(ConditionTypeReady, metav1.ConditionFalse, reasonPendingApproval, "waiting", 1),
	)

	// Act — custom mapping adds new reason; PendingApproval mapping untouched
	summary, err := SummaryFromUnstructured(obj, WithSummaryMapping(customMapping.Merge(defaultMapping)))

	// Assert — PendingApproval behavior intact
	require.NoError(t, err)
	require.Equal(t, StateWaiting, summary.State)
	require.Equal(t, SeverityWarning, summary.Severity)
}

// ── Cross-Consumer Consistency ───────────────────────────────────────────────

func TestSummaryFromUnstructured_SameMappingProducesSameResultAcrossConsumers(t *testing.T) {
	// Arrange — shared mapping defined in platform-types, used by two independent consumers
	const customReason Reason = "ClusterUnreachable"
	sharedMapping := SummaryMapping{
		customReason: {State: StateError, Severity: SeverityError},
	}

	// consumer A (operator) writes conditions and reads summary
	conditionsA := []metav1.Condition{}
	MarkStalled(&conditionsA, 1, customReason, "cluster unreachable")

	obj := newStatusObject(1, 1, conditionsA...)
	summaryFromOperator, err := SummaryFromUnstructured(obj, WithSummaryMapping(sharedMapping))
	require.NoError(t, err)

	// consumer B (service-core) — separate process, same object, same mapping
	summaryFromCore, err := SummaryFromUnstructured(obj, WithSummaryMapping(sharedMapping))
	require.NoError(t, err)

	// Assert — both consumers produce identical summary
	require.Equal(t, summaryFromOperator.KStatus, summaryFromCore.KStatus)
	require.Equal(t, summaryFromOperator.State, summaryFromCore.State)
	require.Equal(t, summaryFromOperator.Severity, summaryFromCore.Severity)
	require.Equal(t, summaryFromOperator.Reason, summaryFromCore.Reason)

	// and both have the expected values
	require.Equal(t, KStatusFailed, summaryFromCore.KStatus)
	require.Equal(t, StateError, summaryFromCore.State)
	require.Equal(t, SeverityError, summaryFromCore.Severity)
	require.Equal(t, customReason, summaryFromCore.Reason)
}

// ── SummaryMapping Merge ─────────────────────────────────────────────────────

func TestSummaryMappingMerge_OverrideReplacesExistingEntry(t *testing.T) {
	// Arrange — override changes Reconciled severity from success to warning
	override := SummaryMapping{
		Reasons.Reconciled: {State: StateReady, Severity: SeverityWarning},
	}

	// Act
	merged := DefaultSummaryMapping().Merge(override)

	// Assert — overridden entry has new values
	entry := merged[Reasons.Reconciled]
	require.Equal(t, SeverityWarning, entry.Severity)
}

func TestSummaryMappingMerge_DefaultEntriesUnaffectedByOverride(t *testing.T) {
	// Arrange — override touches only Reconciled
	override := SummaryMapping{
		Reasons.Reconciled: {State: StateReady, Severity: SeverityWarning},
	}

	// Act
	merged := DefaultSummaryMapping().Merge(override)

	// Assert — other default entries remain intact
	require.Equal(t, SeverityInfo, merged[Reasons.Reconciling].Severity)
	require.Equal(t, StateTerminating, merged[Reasons.Terminating].State)
	require.Equal(t, StateUnknown, merged[Reasons.Unknown].State)
}

func TestSummaryMappingMerge_NewEntryAddedWithoutAffectingDefaults(t *testing.T) {
	// Arrange — override adds brand new reason not in default mapping
	const customReason Reason = "ExternalServiceDown"
	override := SummaryMapping{
		customReason: {State: StateError, Severity: SeverityError},
	}

	// Act
	merged := DefaultSummaryMapping().Merge(override)

	// Assert — new entry present
	entry, ok := merged[customReason]
	require.True(t, ok)
	require.Equal(t, StateError, entry.State)

	// Assert — default entries intact
	require.Equal(t, SeveritySuccess, merged[Reasons.Reconciled].Severity)
	require.Equal(t, SeverityInfo, merged[Reasons.Reconciling].Severity)
}

func TestSummaryMappingMerge_DoesNotMutateOriginalMapping(t *testing.T) {
	// Arrange
	base := DefaultSummaryMapping()
	override := SummaryMapping{
		Reasons.Reconciled: {State: StateReady, Severity: SeverityWarning},
	}

	// Act
	_ = base.Merge(override)

	// Assert — base mapping unchanged
	require.Equal(t, StateReady, base[Reasons.Reconciled].State)
	require.Equal(t, SeveritySuccess, base[Reasons.Reconciled].Severity)
}

// ── Multi-Actor Integration ───────────────────────────────────────────────────

func TestMultiActor_OperatorWritesServiceCoreReads(t *testing.T) {

	// ── operator/pkg/platformstatus ───────────────────────────────────────────

	type clusterConditionType string
	const (
		conditionClusterApproved clusterConditionType = "ClusterApproved"
		conditionClusterJoined   clusterConditionType = "ClusterJoined"
	)

	// domain reasons — defined in operator, imported by consumers
	var clusterReasons = struct {
		PendingApproval Reason
		UserApproved    Reason
		PendingJoin     Reason
	}{
		PendingApproval: "PendingApproval",
		UserApproved:    "UserApproved",
		PendingJoin:     "PendingJoin",
	}

	// summary mapping — defined per resource, merged with SDK defaults
	clusterMapping := SummaryMapping{
		clusterReasons.PendingApproval: {State: StateWaiting, Severity: SeverityWarning},
		clusterReasons.PendingJoin:     {State: StateWaiting, Severity: SeverityWarning},
		clusterReasons.UserApproved:    {State: StateReady, Severity: SeveritySuccess},
	}

	// ── operator: controller ──────────────────────────────────────────────────

	var crdConditions []metav1.Condition
	observedGeneration := int64(2)

	MarkWaiting(&crdConditions, observedGeneration, clusterReasons.PendingApproval, "cluster awaiting approval")

	SetCondition(&crdConditions, NewCondition(
		string(conditionClusterApproved), metav1.ConditionFalse,
		clusterReasons.PendingApproval, "cluster awaiting approval", observedGeneration,
	))
	SetCondition(&crdConditions, NewCondition(
		string(conditionClusterJoined), metav1.ConditionTrue,
		clusterReasons.UserApproved, "cluster joined successfully", observedGeneration,
	))

	crdObject := newStatusObject(observedGeneration, observedGeneration, crdConditions...)

	// ── service-core: use case ────────────────────────────────────────────────

	getClusterStatus := func(obj *unstructured.Unstructured) (Summary, bool, bool, error) {
		summary, err := SummaryFromUnstructured(obj, WithSummaryMapping(clusterMapping))
		if err != nil {
			return Summary{}, false, false, err
		}

		conditions, err := ConditionsFromUnstructured(obj)
		if err != nil {
			return Summary{}, false, false, err
		}

		isApproved := IsConditionTrue(conditions, string(conditionClusterApproved))
		isJoined := IsConditionTrue(conditions, string(conditionClusterJoined))

		return summary, isApproved, isJoined, nil
	}

	summary, isApproved, isJoined, err := getClusterStatus(crdObject)

	// ── frontend: consumes service-core response ──────────────────────────────
	// GET /clusters/:id → { status: { summary, isApproved, isJoined } }
	// summary.State    → badge color
	// summary.Reason   → tooltip / analytics
	// isJoined         → hide join step
	// isApproved       → disable dependent actions

	// ── assert ────────────────────────────────────────────────────────────────

	require.NoError(t, err)

	require.Equal(t, KStatusInProgress, summary.KStatus)
	require.Equal(t, StateWaiting, summary.State)
	require.Equal(t, SeverityWarning, summary.Severity)
	require.Equal(t, clusterReasons.PendingApproval, summary.Reason)

	require.False(t, isApproved, "ClusterApproved should be False — pending approval")
	require.True(t, isJoined, "ClusterJoined should be True — already joined")
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func newStatusObject(generation, observedGeneration int64, conditions ...metav1.Condition) *unstructured.Unstructured {
	rawConditions := make([]interface{}, 0, len(conditions))
	for _, c := range conditions {
		rawConditions = append(rawConditions, map[string]interface{}{
			"type":               c.Type,
			"status":             string(c.Status),
			"reason":             c.Reason,
			"message":            c.Message,
			"observedGeneration": c.ObservedGeneration,
			"lastTransitionTime": c.LastTransitionTime.Format("2006-01-02T15:04:05Z"),
		})
	}

	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "platform.totvs.app/v1",
		"kind":       "Example",
		"metadata": map[string]interface{}{
			"name":       "example",
			"generation": generation,
		},
		"status": map[string]interface{}{
			"observedGeneration": observedGeneration,
			"conditions":         rawConditions,
		},
	}}
}

func condition(conditionType string, status metav1.ConditionStatus, reason Reason, message string, observedGeneration int64) metav1.Condition {
	return metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: observedGeneration,
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
}
