package status

import (
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
)

type KStatus string

const (
	KStatusInProgress  KStatus = KStatus(kstatus.InProgressStatus)
	KStatusFailed      KStatus = KStatus(kstatus.FailedStatus)
	KStatusCurrent     KStatus = KStatus(kstatus.CurrentStatus)
	KStatusTerminating KStatus = KStatus(kstatus.TerminatingStatus)
	KStatusNotFound    KStatus = KStatus(kstatus.NotFoundStatus)
	KStatusUnknown     KStatus = KStatus(kstatus.UnknownStatus)
)

type State string

const (
	StateReady       State = "ready"
	StateProgressing State = "progressing"
	StateWaiting     State = "waiting"
	StateWarning     State = "warning"
	StateError       State = "error"
	StateTerminating State = "terminating"
	StateNotFound    State = "notFound"
	StateUnknown     State = "unknown"
)

type Severity string

const (
	SeveritySuccess Severity = "success"
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Reason is a stable machine-readable reason code used as a key in SummaryMapping.
type Reason string

// sdkCoreReasons are used internally by the SDK Mark* helpers.
type sdkCoreReasons struct {
	Reconciled  Reason // ~200 OK
	Reconciling Reason // ~102 Processing
	Terminating Reason // ~102 Processing (deletion)
	Unknown     Reason // ~520 Unknown
}

// sdkCommonReasons are suggested vocabulary for controllers — not used internally by the SDK.
type sdkCommonReasons struct {
	InvalidConfiguration  Reason // ~400 Bad Request
	DependencyNotFound    Reason // ~404 Not Found
	DependencyUnavailable Reason // ~503 Service Unavailable
	Conflict              Reason // ~409 Conflict
	PreconditionNotMet    Reason // ~428 Precondition Required
	PermissionDenied      Reason // ~403 Forbidden
	Timeout               Reason // ~408 Request Timeout
}

type sdkReasons struct {
	sdkCoreReasons
	sdkCommonReasons
}

// Reasons is the SDK built-in reason vocabulary.
// Do not modify — treat as read-only. Type Reasons. in your IDE to see all available reasons.
var Reasons = sdkReasons{
	sdkCoreReasons: sdkCoreReasons{
		Reconciled:  "Reconciled",
		Reconciling: "Reconciling",
		Terminating: "Terminating",
		Unknown:     "Unknown",
	},
	sdkCommonReasons: sdkCommonReasons{
		InvalidConfiguration:  "InvalidConfiguration",
		DependencyNotFound:    "DependencyNotFound",
		DependencyUnavailable: "DependencyUnavailable",
		Conflict:              "Conflict",
		PreconditionNotMet:    "PreconditionNotMet",
		PermissionDenied:      "PermissionDenied",
		Timeout:               "Timeout",
	},
}

const (
	ConditionTypeReady       = "Ready"
	ConditionTypeReconciling = string(kstatus.ConditionReconciling)
	ConditionTypeStalled     = string(kstatus.ConditionStalled)
)

// Summary is the normalized status contract for API and UI consumers.
type Summary struct {
	KStatus  KStatus  `json:"kstatus"`
	State    State    `json:"state"`
	Severity Severity `json:"severity"`
	Reason   Reason   `json:"reason,omitempty"`
	Message  string   `json:"message,omitempty"`
}
