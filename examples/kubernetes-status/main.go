package main

import (
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	status "github.com/totvs/go-sdk/kubernetes/status"
)

// writeExample demonstrates operator writing KStatus-compliant conditions.
func writeExample() {
	var conditions []metav1.Condition
	gen := int64(1)

	// resource reconciling
	status.MarkReconciling(&conditions, gen, status.Reasons.Reconciling, "installing helm chart")
	log.Printf("  MarkReconciling → conditions: %d", len(conditions))

	// resource ready
	status.MarkReady(&conditions, gen, status.Reasons.Reconciled, "helm chart installed")
	log.Printf("  MarkReady → conditions: %d (Reconciling removed)", len(conditions))

	// resource stalled — dependency not found
	status.MarkStalled(&conditions, gen, status.Reasons.DependencyNotFound, "driver CRD not found")
	log.Printf("  MarkStalled → conditions: %d (Stalled=True added)", len(conditions))

	// resource waiting — domain reason
	status.MarkWaiting(&conditions, gen, "PendingApproval", "cluster awaiting approval")
	log.Printf("  MarkWaiting → conditions: %d (Reconciling+Stalled removed)", len(conditions))
}

// readDefaultExample demonstrates reading Summary with no custom mapping.
func readDefaultExample() {
	conditions := []metav1.Condition{}
	status.MarkReady(&conditions, 1, status.Reasons.Reconciled, "reconciled")

	obj := buildObject(1, 1, conditions...)
	summary, err := status.SummaryFromUnstructured(obj)
	if err != nil {
		log.Fatalf("SummaryFromUnstructured: %v", err)
	}

	log.Printf("  KStatus:  %s", summary.KStatus)
	log.Printf("  State:    %s", summary.State)
	log.Printf("  Severity: %s", summary.Severity)
	log.Printf("  Reason:   %s", summary.Reason)
}

// readWithCustomMappingExample demonstrates injecting domain-specific mapping.
func readWithCustomMappingExample() {
	// domain reasons — defined in operator/pkg/platformstatus
	const reasonPendingApproval status.Reason = "PendingApproval"

	// summary mapping — defined per resource, merged with SDK defaults
	clusterMapping := status.SummaryMapping{
		reasonPendingApproval: {State: status.StateWaiting, Severity: status.SeverityWarning},
	}

	// operator writes
	conditions := []metav1.Condition{}
	status.MarkWaiting(&conditions, 1, reasonPendingApproval, "cluster awaiting approval")
	obj := buildObject(1, 1, conditions...)

	// service-core reads with custom mapping
	summary, err := status.SummaryFromUnstructured(obj, status.WithSummaryMapping(clusterMapping))
	if err != nil {
		log.Fatalf("SummaryFromUnstructured: %v", err)
	}

	log.Printf("  KStatus:  %s", summary.KStatus)
	log.Printf("  State:    %s", summary.State)    // waiting (from mapping)
	log.Printf("  Severity: %s", summary.Severity) // warning (from mapping)
	log.Printf("  Reason:   %s", summary.Reason)
}

// commonReasonsExample demonstrates common SDK reasons inferred automatically.
func commonReasonsExample() {
	// terminal failures → MarkStalled (KStatus=Failed)
	stalledCases := []struct {
		label  string
		reason status.Reason
		msg    string
	}{
		{"DependencyNotFound", status.Reasons.DependencyNotFound, "driver CRD missing"},
		{"DependencyUnavailable", status.Reasons.DependencyUnavailable, "database not ready"},
		{"InvalidConfiguration", status.Reasons.InvalidConfiguration, "invalid helm values"},
		{"PermissionDenied", status.Reasons.PermissionDenied, "SA missing RBAC"},
		{"Timeout", status.Reasons.Timeout, "webhook timed out"},
		{"Conflict", status.Reasons.Conflict, "resource locked"},
	}

	for _, c := range stalledCases {
		conditions := []metav1.Condition{}
		status.MarkStalled(&conditions, 1, c.reason, c.msg)
		obj := buildObject(1, 1, conditions...)

		summary, err := status.SummaryFromUnstructured(obj)
		if err != nil {
			log.Fatalf("SummaryFromUnstructured: %v", err)
		}
		log.Printf("  %-25s → state:%-15s severity:%s", c.label, summary.State, summary.Severity)
	}

	// intentional wait → MarkWaiting (KStatus=InProgress, state=waiting)
	conditions := []metav1.Condition{}
	status.MarkWaiting(&conditions, 1, status.Reasons.PreconditionNotMet, "TLS cert not issued")
	obj := buildObject(1, 1, conditions...)

	summary, err := status.SummaryFromUnstructured(obj)
	if err != nil {
		log.Fatalf("SummaryFromUnstructured: %v", err)
	}
	log.Printf("  %-25s → state:%-15s severity:%s", "PreconditionNotMet", summary.State, summary.Severity)
}

// domainConditionExample demonstrates domain conditions alongside KStatus conditions.
func domainConditionExample() {
	conditions := []metav1.Condition{}
	gen := int64(1)

	// KStatus signal
	status.MarkWaiting(&conditions, gen, "PendingApproval", "awaiting approval")

	// domain condition — business state, does not affect KStatus
	domainCond := status.NewCondition("ClusterApproved", metav1.ConditionFalse, "PendingApproval", "not yet approved", gen)
	status.SetCondition(&conditions, domainCond)

	log.Printf("  conditions total: %d", len(conditions))
	log.Printf("  ClusterApproved: %v", status.IsConditionFalse(conditions, domainCond.Type))
	log.Printf("  Ready: %v (IsConditionFalse)", status.IsConditionFalse(conditions, status.ConditionTypeReady))
}

// notFoundExample demonstrates NotFoundSummary for missing resources.
func notFoundExample() {
	summary := status.NotFoundSummary(status.Reasons.Unknown, "resource not found in cluster")
	log.Printf("  KStatus:  %s", summary.KStatus)
	log.Printf("  State:    %s", summary.State)
	log.Printf("  Severity: %s", summary.Severity)
}

// computeExample demonstrates raw KStatus computation without Summary.
func computeExample() {
	conditions := []metav1.Condition{}
	status.MarkStalled(&conditions, 1, status.Reasons.DependencyNotFound, "driver missing")
	obj := buildObject(1, 1, conditions...)

	ks, err := status.ComputeFromUnstructured(obj)
	if err != nil {
		log.Fatalf("Compute: %v", err)
	}
	log.Printf("  KStatus: %s", ks) // Failed
}

func main() {
	log.Println("=== Kubernetes Status SDK Examples ===")
	log.Println()

	log.Println("1. Write helpers (operator/controller)")
	writeExample()

	log.Println("\n2. Read Summary — default mapping")
	readDefaultExample()

	log.Println("\n3. Read Summary — custom domain mapping")
	readWithCustomMappingExample()

	log.Println("\n4. Common SDK reasons — automatic inference")
	commonReasonsExample()

	log.Println("\n5. Domain conditions alongside KStatus")
	domainConditionExample()

	log.Println("\n6. NotFoundSummary")
	notFoundExample()

	log.Println("\n7. Raw KStatus computation (no Summary)")
	computeExample()

	log.Println("\n=== All examples completed ===")
}

func buildObject(generation, observedGeneration int64, conditions ...metav1.Condition) *unstructured.Unstructured {
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
