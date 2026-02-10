package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mocks ---

type MockCollector struct {
	mock.Mock
}

func (m *MockCollector) Collect(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

type MockProcessor struct {
	mock.Mock
}

func (m *MockProcessor) Process(ctx context.Context, items []string) (map[string]int, error) {
	args := m.Called(ctx, items)
	return args.Get(0).(map[string]int), args.Error(1)
}

type MockTransmitter struct {
	mock.Mock
}

func (m *MockTransmitter) Transmit(ctx context.Context, report Report) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

type stubPostProcessor struct {
	fn func(Report) Report
}

func (s *stubPostProcessor) Process(report Report) Report {
	return s.fn(report)
}

// --- Suite ---

func TestPipelineTestSuite(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite
}

func (s *PipelineTestSuite) TestShouldExecuteSuccessfully() {
	// Arrange
	mockCollector := new(MockCollector)
	mockProcessor := new(MockProcessor)
	mockTransmitter := new(MockTransmitter)

	mockCollector.On("Collect", mock.Anything).Return([]string{"item1", "item2"}, nil)
	mockProcessor.On("Process", mock.Anything, []string{"item1", "item2"}).Return(map[string]int{"count": 2}, nil)
	mockTransmitter.On("Transmit", mock.Anything, mock.MatchedBy(func(report Report) bool {
		return report.Source == "test-cluster" && report.Analyses["test-step"] != nil
	})).Return(nil)

	p := New("test-cluster", mockTransmitter)
	p.AddStep(NewStep("test-step", mockCollector, mockProcessor))

	// Act
	err := p.Run(context.Background())

	// Assert
	s.NoError(err)
	mockCollector.AssertExpectations(s.T())
	mockProcessor.AssertExpectations(s.T())
	mockTransmitter.AssertExpectations(s.T())
}

func (s *PipelineTestSuite) TestShouldFailWhenCollectorReturnsError() {
	// Arrange
	mockCollector := new(MockCollector)
	mockProcessor := new(MockProcessor)
	mockTransmitter := new(MockTransmitter)

	mockCollector.On("Collect", mock.Anything).Return([]string{}, errors.New("collector error"))

	p := New("test-cluster", mockTransmitter)
	p.AddStep(NewStep("test-step", mockCollector, mockProcessor))

	// Act
	err := p.Run(context.Background())

	// Assert
	s.Error(err)
	s.Contains(err.Error(), "collect failed")
	mockProcessor.AssertNotCalled(s.T(), "Process")
	mockTransmitter.AssertNotCalled(s.T(), "Transmit")
}

func (s *PipelineTestSuite) TestShouldFailWhenProcessorReturnsError() {
	// Arrange
	mockCollector := new(MockCollector)
	mockProcessor := new(MockProcessor)
	mockTransmitter := new(MockTransmitter)

	mockCollector.On("Collect", mock.Anything).Return([]string{"item1"}, nil)
	mockProcessor.On("Process", mock.Anything, []string{"item1"}).Return(map[string]int{}, errors.New("processor error"))

	p := New("test-cluster", mockTransmitter)
	p.AddStep(NewStep("test-step", mockCollector, mockProcessor))

	// Act
	err := p.Run(context.Background())

	// Assert
	s.Error(err)
	s.Contains(err.Error(), "process failed")
	mockTransmitter.AssertNotCalled(s.T(), "Transmit")
}

func (s *PipelineTestSuite) TestShouldFailWhenTransmitterReturnsError() {
	// Arrange
	mockCollector := new(MockCollector)
	mockProcessor := new(MockProcessor)
	mockTransmitter := new(MockTransmitter)

	mockCollector.On("Collect", mock.Anything).Return([]string{"item1"}, nil)
	mockProcessor.On("Process", mock.Anything, []string{"item1"}).Return(map[string]int{"count": 1}, nil)
	mockTransmitter.On("Transmit", mock.Anything, mock.Anything).Return(errors.New("transmission error"))

	p := New("test-cluster", mockTransmitter)
	p.AddStep(NewStep("test-step", mockCollector, mockProcessor))

	// Act
	err := p.Run(context.Background())

	// Assert
	s.Error(err)
	s.Contains(err.Error(), "transmission failed")
}

func (s *PipelineTestSuite) TestShouldExecuteMultipleSteps() {
	// Arrange
	mockCollector1 := new(MockCollector)
	mockProcessor1 := new(MockProcessor)
	mockCollector2 := new(MockCollector)
	mockProcessor2 := new(MockProcessor)
	mockTransmitter := new(MockTransmitter)

	mockCollector1.On("Collect", mock.Anything).Return([]string{"item1"}, nil)
	mockProcessor1.On("Process", mock.Anything, []string{"item1"}).Return(map[string]int{"count": 1}, nil)
	mockCollector2.On("Collect", mock.Anything).Return([]string{"item2", "item3"}, nil)
	mockProcessor2.On("Process", mock.Anything, []string{"item2", "item3"}).Return(map[string]int{"count": 2}, nil)
	mockTransmitter.On("Transmit", mock.Anything, mock.MatchedBy(func(report Report) bool {
		return len(report.Analyses) == 2 && report.Analyses["step1"] != nil && report.Analyses["step2"] != nil
	})).Return(nil)

	p := New("test-cluster", mockTransmitter)
	p.AddStep(NewStep("step1", mockCollector1, mockProcessor1)).
		AddStep(NewStep("step2", mockCollector2, mockProcessor2))

	// Act
	err := p.Run(context.Background())

	// Assert
	s.NoError(err)
	mockCollector1.AssertExpectations(s.T())
	mockProcessor1.AssertExpectations(s.T())
	mockCollector2.AssertExpectations(s.T())
	mockProcessor2.AssertExpectations(s.T())
	mockTransmitter.AssertExpectations(s.T())
}

func (s *PipelineTestSuite) TestShouldSkipWhenNoStepsRegistered() {
	// Arrange
	mockTransmitter := new(MockTransmitter)
	p := New("test-cluster", mockTransmitter)

	// Act
	err := p.Run(context.Background())

	// Assert
	s.NoError(err)
	mockTransmitter.AssertNotCalled(s.T(), "Transmit")
}

func (s *PipelineTestSuite) TestShouldEnrichReportViaPostProcessor() {
	// Arrange
	mockCollector := new(MockCollector)
	mockProcessor := new(MockProcessor)
	mockTransmitter := new(MockTransmitter)

	mockCollector.On("Collect", mock.Anything).Return([]string{"item1"}, nil)
	mockProcessor.On("Process", mock.Anything, []string{"item1"}).Return(map[string]int{"count": 1}, nil)
	mockTransmitter.On("Transmit", mock.Anything, mock.MatchedBy(func(report Report) bool {
		return report.Metrics["enriched"] == 1.0
	})).Return(nil)

	pp := &stubPostProcessor{
		fn: func(report Report) Report {
			report.Metrics["enriched"] = 1.0
			return report
		},
	}

	p := New("test-cluster", mockTransmitter)
	p.AddStep(NewStep("test-step", mockCollector, mockProcessor)).
		PostProcess(pp)

	// Act
	err := p.Run(context.Background())

	// Assert
	s.NoError(err)
	mockTransmitter.AssertExpectations(s.T())
}

func (s *PipelineTestSuite) TestShouldPanicOnNilTransmitter() {
	s.Panics(func() {
		New("test-cluster", nil)
	})
}

func (s *PipelineTestSuite) TestShouldPanicOnNilCollector() {
	// Arrange
	mockTransmitter := new(MockTransmitter)
	mockProcessor := new(MockProcessor)
	_ = New("test-cluster", mockTransmitter)

	// Act & Assert
	s.Panics(func() {
		NewStep("test-step", nil, mockProcessor)
	})
}

func (s *PipelineTestSuite) TestShouldPanicOnNilProcessor() {
	// Arrange
	mockTransmitter := new(MockTransmitter)
	mockCollector := new(MockCollector)
	_ = New("test-cluster", mockTransmitter)

	// Act & Assert
	s.Panics(func() {
		NewStep[string, map[string]int]("test-step", mockCollector, nil)
	})
}

func (s *PipelineTestSuite) TestShouldPanicOnNilPostProcessor() {
	// Arrange
	mockTransmitter := new(MockTransmitter)
	p := New("test-cluster", mockTransmitter)

	// Act & Assert
	s.Panics(func() {
		p.PostProcess(nil)
	})
}
