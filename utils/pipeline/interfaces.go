package pipeline

import "context"

// Collector gathers raw resources from a data source.
type Collector[T any] interface {
	Collect(ctx context.Context) ([]T, error)
}

// Processor transforms collected items into a typed analysis result.
type Processor[T any, R any] interface {
	Process(ctx context.Context, items []T) (R, error)
}

// PostProcessor enriches the aggregated report after all steps complete.
type PostProcessor interface {
	Process(report Report) Report
}

// Transmitter sends the final report to a destination.
type Transmitter interface {
	Transmit(ctx context.Context, report Report) error
}
