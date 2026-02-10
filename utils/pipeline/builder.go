package pipeline

import (
	"sync"
	"time"
)

// ReportBuilder aggregates multiple analyses into a final report (thread-safe).
type ReportBuilder struct {
	mu     sync.Mutex
	source string
	data   map[string]interface{}
}

func NewReportBuilder(source string) *ReportBuilder {
	return &ReportBuilder{
		source: source,
		data:   make(map[string]interface{}),
	}
}

func (b *ReportBuilder) Add(name string, analysis interface{}) *ReportBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.data[name] = analysis
	return b
}

// Build produces the final Report.
func (b *ReportBuilder) Build() Report {
	b.mu.Lock()
	defer b.mu.Unlock()

	return Report{
		Source:    b.source,
		Timestamp: time.Now().UTC(),
		Analyses:  b.data,
		Metrics:   make(map[string]float64),
	}
}
