package transmitter

import "github.com/totvs/go-sdk/utils/pipeline"

// CRDMapper maps a pipeline Report to a CRD object.
type CRDMapper[T any] interface {
	// NewObject returns an empty instance for Get.
	NewObject() T
	// MapToCreate prepares the object for initial creation.
	MapToCreate(clusterName string) T
	// MapToStatus updates the object status with report data.
	MapToStatus(obj T, report pipeline.Report)
}
