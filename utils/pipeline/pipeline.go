package pipeline

import (
	"context"
	"fmt"

	logger "github.com/totvs/go-sdk/log"
	"golang.org/x/sync/errgroup"
)

// Pipeline runs multiple steps in parallel and transmits the aggregated report.
type Pipeline struct {
	source         string
	steps          []step
	postProcessors []PostProcessor
	transmitter    Transmitter
}

// step wraps the execution of a Collector+Processor pair.
type step struct {
	name    string
	execute StepFunc
}

// New creates a new Pipeline.
func New(source string, transmitter Transmitter) *Pipeline {
	if transmitter == nil {
		panic("transmitter cannot be nil")
	}

	return &Pipeline{
		source:      source,
		steps:       make([]step, 0),
		transmitter: transmitter,
	}
}

// AddStep registers a step in the pipeline.
func (p *Pipeline) AddStep(name string, fn StepFunc) *Pipeline {
	p.steps = append(p.steps, step{
		name:    name,
		execute: fn,
	})
	return p
}

// Add registers a step using StepConfig.
func (p *Pipeline) Add(stepConfig StepConfig) *Pipeline {
	return p.AddStep(stepConfig.Name, stepConfig.Func)
}

// PostProcess registers a PostProcessor to run after Build.
func (p *Pipeline) PostProcess(pp PostProcessor) *Pipeline {
	if pp == nil {
		panic("postProcessor cannot be nil")
	}
	p.postProcessors = append(p.postProcessors, pp)
	return p
}

// Run executes all steps in parallel, aggregates results and transmits.
func (p *Pipeline) Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Info().Msgf("[Pipeline] Starting pipeline for source '%s' with %d steps", p.source, len(p.steps))

	if len(p.steps) == 0 {
		l.Warn().Msg("[Pipeline] No steps registered, skipping execution")
		return nil
	}

	builder := NewReportBuilder(p.source)
	g, gCtx := errgroup.WithContext(ctx)

	for _, s := range p.steps {
		s := s
		g.Go(func() error {
			return s.execute(gCtx, builder)
		})
	}

	if err := g.Wait(); err != nil {
		l.Error(err).Msg("[Pipeline] One or more steps failed")
		return err
	}

	l.Info().Msg("[Pipeline] All steps completed successfully")

	report := builder.Build()

	for _, pp := range p.postProcessors {
		report = pp.Process(report)
	}

	// Use the original context (not the errgroup's which may be cancelled)
	l.Debug().Msg("[Pipeline] Transmitting report...")
	if err := p.transmitter.Transmit(ctx, report); err != nil {
		l.Error(err).Msg("[Pipeline] Transmission failed")
		return fmt.Errorf("transmission failed: %w", err)
	}

	l.Info().Msgf("[Pipeline] Pipeline completed successfully for source '%s'", p.source)
	return nil
}
