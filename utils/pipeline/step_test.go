package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestStepTestSuite(t *testing.T) {
	suite.Run(t, new(StepTestSuite))
}

type StepTestSuite struct {
	suite.Suite
}

func (s *StepTestSuite) TestShouldReturnNameAndFunc() {
	// Arrange
	collector := &fakeCollector{items: []string{"a", "b"}}
	processor := &fakeProcessor{result: "ok"}

	// Act
	name, fn := NewStep("test-step", collector, processor)

	// Assert
	s.Equal("test-step", name)
	s.NotNil(fn)
}

func (s *StepTestSuite) TestShouldCollectProcessAndAddToBuilder() {
	// Arrange
	collector := &fakeCollector{items: []string{"a", "b"}}
	processor := &fakeProcessor{result: "processed"}
	_, fn := NewStep("my-step", collector, processor)
	builder := NewReportBuilder("cluster1")

	// Act
	err := fn(context.Background(), builder)

	// Assert
	s.NoError(err)
	report := builder.Build()
	s.Equal("processed", report.Analyses["my-step"])
}

func (s *StepTestSuite) TestShouldReturnErrorWhenCollectFails() {
	// Arrange
	collector := &fakeCollector{err: errors.New("collect failed")}
	processor := &fakeProcessor{result: "unused"}
	_, fn := NewStep("failing-step", collector, processor)
	builder := NewReportBuilder("cluster1")

	// Act
	err := fn(context.Background(), builder)

	// Assert
	s.Error(err)
	s.Contains(err.Error(), "collect failed")
}

func (s *StepTestSuite) TestShouldReturnErrorWhenProcessFails() {
	// Arrange
	collector := &fakeCollector{items: []string{"a"}}
	processor := &fakeProcessor{err: errors.New("process failed")}
	_, fn := NewStep("failing-step", collector, processor)
	builder := NewReportBuilder("cluster1")

	// Act
	err := fn(context.Background(), builder)

	// Assert
	s.Error(err)
	s.Contains(err.Error(), "process failed")
}

func (s *StepTestSuite) TestShouldPanicOnNilCollector() {
	s.Panics(func() {
		NewStep[string, string]("step", nil, &fakeProcessor{})
	})
}

func (s *StepTestSuite) TestShouldPanicOnNilProcessor() {
	s.Panics(func() {
		NewStep[string, string]("step", &fakeCollector{}, nil)
	})
}

// --- Fakes ---

type fakeCollector struct {
	items []string
	err   error
}

func (c *fakeCollector) Collect(_ context.Context) ([]string, error) {
	return c.items, c.err
}

type fakeProcessor struct {
	result string
	err    error
}

func (p *fakeProcessor) Process(_ context.Context, _ []string) (string, error) {
	return p.result, p.err
}
