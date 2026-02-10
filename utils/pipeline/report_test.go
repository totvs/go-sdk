package pipeline

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

func TestReportTestSuite(t *testing.T) {
	suite.Run(t, new(ReportTestSuite))
}

type ReportTestSuite struct {
	suite.Suite
}

func (s *ReportTestSuite) TestShouldMarshalToJSON() {
	// Arrange
	report := Report{
		Source:    "spoke1",
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Analyses:  map[string]interface{}{"pods": map[string]int{"total": 5}},
		Metrics:   map[string]float64{"ratio": 0.8},
	}

	// Act
	data, err := json.Marshal(report)

	// Assert
	s.NoError(err)
	s.Contains(string(data), `"source":"spoke1"`)
	s.Contains(string(data), `"ratio":0.8`)
}

func (s *ReportTestSuite) TestShouldOmitEmptyAnalysesAndMetrics() {
	// Arrange
	report := Report{
		Source:    "spoke1",
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Act
	data, err := json.Marshal(report)

	// Assert
	s.NoError(err)
	s.NotContains(string(data), `"analyses"`)
	s.NotContains(string(data), `"metrics"`)
}

func (s *ReportTestSuite) TestShouldUnmarshalFromJSON() {
	// Arrange
	raw := `{"source":"spoke1","analyses":{"pods":{"total":3}},"metrics":{"ratio":0.5}}`

	// Act
	var report Report
	err := json.Unmarshal([]byte(raw), &report)

	// Assert
	s.NoError(err)
	s.Equal("spoke1", report.Source)
	s.Contains(report.Analyses, "pods")
	s.Equal(0.5, report.Metrics["ratio"])
}
