package pipeline

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestReportBuilderTestSuite(t *testing.T) {
	suite.Run(t, new(ReportBuilderTestSuite))
}

type ReportBuilderTestSuite struct {
	suite.Suite
}

func (s *ReportBuilderTestSuite) TestShouldReturnSelfOnAdd() {
	// Arrange
	builder := NewReportBuilder("test-cluster")

	// Act
	result := builder.Add("test-key", map[string]int{"total": 10})

	// Assert
	s.NotNil(result)
	s.Same(builder, result)
}

func (s *ReportBuilderTestSuite) TestShouldBuildReportWithAnalysis() {
	// Arrange
	builder := NewReportBuilder("test-cluster")
	testAnalysis := map[string]int{"total": 10, "running": 8}

	// Act
	builder.Add("test-key", testAnalysis)
	report := builder.Build()

	// Assert
	s.Equal("test-cluster", report.Source)
	s.NotZero(report.Timestamp)
	s.Contains(report.Analyses, "test-key")
	s.Equal(testAnalysis, report.Analyses["test-key"])
}

func (s *ReportBuilderTestSuite) TestShouldBuildReportWithMultipleAnalyses() {
	// Arrange
	builder := NewReportBuilder("test-cluster")
	podAnalysis := map[string]int{"total": 10}
	nodeAnalysis := struct{ Total int }{Total: 3}

	// Act
	builder.Add("pods", podAnalysis)
	builder.Add("nodes", nodeAnalysis)
	report := builder.Build()

	// Assert
	s.Len(report.Analyses, 2)
	s.Contains(report.Analyses, "pods")
	s.Contains(report.Analyses, "nodes")
}

func (s *ReportBuilderTestSuite) TestShouldBuildReportWithEmptyMetrics() {
	// Arrange
	builder := NewReportBuilder("test-cluster")
	builder.Add("test-key", map[string]int{"total": 10, "running": 8})

	// Act
	report := builder.Build()

	// Assert
	s.NotNil(report.Metrics)
	s.Empty(report.Metrics)
}

func (s *ReportBuilderTestSuite) TestShouldBeThreadSafe() {
	// Arrange
	builder := NewReportBuilder("test-cluster")
	done := make(chan bool)

	// Act
	go func() {
		builder.Add("pods", map[string]int{"total": 10})
		done <- true
	}()
	go func() {
		builder.Add("nodes", struct{ Total int }{Total: 3})
		done <- true
	}()
	<-done
	<-done
	report := builder.Build()

	// Assert
	s.Len(report.Analyses, 2)
}
