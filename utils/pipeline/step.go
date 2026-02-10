package pipeline

import (
	"context"
	"fmt"

	logger "github.com/totvs/go-sdk/log"
)

// StepFunc is the execution signature for a pipeline step.
type StepFunc func(ctx context.Context, builder *ReportBuilder) error

// StepConfig holds the configuration for a pipeline step.
type StepConfig struct {
	Name string
	Func StepFunc
}

// NewStep creates a type-safe StepFunc from a Collector and Processor.
func NewStep[T any, R any](
	name string,
	collector Collector[T],
	processor Processor[T, R],
) (string, StepFunc) {
	if collector == nil {
		panic(fmt.Sprintf("collector for step '%s' cannot be nil", name))
	}
	if processor == nil {
		panic(fmt.Sprintf("processor for step '%s' cannot be nil", name))
	}

	fn := func(ctx context.Context, builder *ReportBuilder) error {
		l := logger.FromContext(ctx)
		l.Debug().Msgf("[Pipeline] Step '%s' starting", name)

		data, err := collector.Collect(ctx)
		if err != nil {
			l.Error(err).Msgf("[Pipeline] Step '%s' collect failed", name)
			return fmt.Errorf("step '%s' collect failed: %w", name, err)
		}
		l.Debug().Msgf("[Pipeline] Step '%s' collected %d items", name, len(data))

		result, err := processor.Process(ctx, data)
		if err != nil {
			l.Error(err).Msgf("[Pipeline] Step '%s' process failed", name)
			return fmt.Errorf("step '%s' process failed: %w", name, err)
		}
		l.Debug().Msgf("[Pipeline] Step '%s' processed successfully", name)

		builder.Add(name, result)
		return nil
	}

	return name, fn
}
