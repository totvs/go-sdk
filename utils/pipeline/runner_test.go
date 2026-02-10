package pipeline

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// --- Suite ---

func TestPeriodicRunnerTestSuite(t *testing.T) {
	suite.Run(t, new(PeriodicRunnerTestSuite))
}

type PeriodicRunnerTestSuite struct {
	suite.Suite
}

func (s *PeriodicRunnerTestSuite) newCountingPipeline() (*Pipeline, *atomic.Int32) {
	var count atomic.Int32

	transmitter := &countingTransmitter{count: &count}
	p := New("test-cluster", transmitter)
	p.AddStep("counting-step", func(ctx context.Context, builder *ReportBuilder) error {
		builder.Add("test", "data")
		return nil
	})

	return p, &count
}

func (s *PeriodicRunnerTestSuite) TestShouldRunImmediately() {
	// Arrange
	p, count := s.newCountingPipeline()
	runner := NewPeriodicRunner(1 * time.Hour) // intervalo longo para não disparar
	ctx, cancel := context.WithCancel(context.Background())

	// Act
	go runner.Run(ctx, p)
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Assert
	s.Equal(int32(1), count.Load(), "pipeline should have run exactly once (immediately)")
}

func (s *PeriodicRunnerTestSuite) TestShouldRunPeriodically() {
	// Arrange
	p, count := s.newCountingPipeline()
	runner := NewPeriodicRunner(50 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	// Act
	go runner.Run(ctx, p)
	time.Sleep(180 * time.Millisecond) // immediate + ~2-3 ticks
	cancel()

	// Assert
	runs := count.Load()
	s.GreaterOrEqual(runs, int32(3), "pipeline should have run at least 3 times (immediate + 2 ticks)")
}

func (s *PeriodicRunnerTestSuite) TestShouldStopOnContextCancel() {
	// Arrange
	p, count := s.newCountingPipeline()
	runner := NewPeriodicRunner(50 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	// Act
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx, p)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	err := <-done

	// Assert
	s.NoError(err)

	runsAtCancel := count.Load()
	time.Sleep(100 * time.Millisecond)
	runsAfterCancel := count.Load()

	s.Equal(runsAtCancel, runsAfterCancel, "pipeline should not run after context cancel")
}

func (s *PeriodicRunnerTestSuite) TestShouldRunMultiplePipelines() {
	// Arrange
	p1, count1 := s.newCountingPipeline()
	p2, count2 := s.newCountingPipeline()
	runner := NewPeriodicRunner(1 * time.Hour)
	ctx, cancel := context.WithCancel(context.Background())

	// Act
	go runner.Run(ctx, p1, p2)
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Assert
	s.Equal(int32(1), count1.Load(), "pipeline 1 should have run once")
	s.Equal(int32(1), count2.Load(), "pipeline 2 should have run once")
}

func (s *PeriodicRunnerTestSuite) TestShouldSkipNilPipelines() {
	// Arrange
	p, count := s.newCountingPipeline()
	runner := NewPeriodicRunner(1 * time.Hour)
	ctx, cancel := context.WithCancel(context.Background())

	// Act
	go runner.Run(ctx, nil, p, nil)
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Assert
	s.Equal(int32(1), count.Load(), "non-nil pipeline should have run once")
}

// --- Helper ---

type countingTransmitter struct {
	count *atomic.Int32
}

func (t *countingTransmitter) Transmit(_ context.Context, _ Report) error {
	t.count.Add(1)
	return nil
}
