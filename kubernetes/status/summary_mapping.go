package status

// SummaryRule defines how a Reason maps to Summary presentation fields.
type SummaryRule struct {
	State    State
	Severity Severity
}

// SummaryMapping maps Reason codes to their Summary presentation rules.
type SummaryMapping map[Reason]SummaryRule

// DefaultSummaryMapping returns the SDK built-in reason-to-summary mapping.
// Domain-specific reasons belong in the consumer's own SummaryMapping, merged via WithSummaryMapping.
func DefaultSummaryMapping() SummaryMapping {
	return coreSummaryMapping().Merge(commonSummaryMapping())
}

// coreSummaryMapping covers reasons used internally by the SDK Mark* helpers.
func coreSummaryMapping() SummaryMapping {
	return SummaryMapping{
		Reasons.Reconciled:  {State: StateReady, Severity: SeveritySuccess},
		Reasons.Reconciling: {State: StateProgressing, Severity: SeverityInfo},
		Reasons.Terminating: {State: StateTerminating, Severity: SeverityWarning},
		Reasons.Unknown:     {State: StateUnknown, Severity: SeverityWarning},
	}
}

// commonSummaryMapping covers suggested vocabulary reasons — mirrors sdkCommonReasons.
func commonSummaryMapping() SummaryMapping {
	return SummaryMapping{
		Reasons.InvalidConfiguration:  {State: StateError, Severity: SeverityError},
		Reasons.DependencyNotFound:    {State: StateError, Severity: SeverityError},
		Reasons.DependencyUnavailable: {State: StateError, Severity: SeverityWarning},
		Reasons.Conflict:              {State: StateError, Severity: SeverityWarning},
		Reasons.PreconditionNotMet:    {State: StateWaiting, Severity: SeverityWarning},
		Reasons.PermissionDenied:      {State: StateError, Severity: SeverityError},
		Reasons.Timeout:               {State: StateError, Severity: SeverityWarning},
	}
}

// Merge returns a new SummaryMapping with override entries applied on top of m.
// The original mapping is not mutated.
func (m SummaryMapping) Merge(override SummaryMapping) SummaryMapping {
	merged := make(SummaryMapping, len(m)+len(override))
	for reason, rule := range m {
		merged[reason] = rule
	}
	for reason, rule := range override {
		merged[reason] = rule
	}
	return merged
}
