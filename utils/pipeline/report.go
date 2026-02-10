package pipeline

import "time"

// Report holds the aggregated analysis data produced by a pipeline.
type Report struct {
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Analyses  map[string]interface{} `json:"analyses,omitempty"`
	Metrics   map[string]float64     `json:"metrics,omitempty"`
}
