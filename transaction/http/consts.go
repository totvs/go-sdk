package transaction_http

// Public constants for trace header and field names used across projects.
const (
	// TraceIDHeader is the HTTP header used to carry the request trace id.
	TraceIDHeader = "X-Request-Id"
	// TraceIDCorrelationHeader is the alternate header name often used for correlation ids.
	TraceIDCorrelationHeader = "X-Correlation-Id"
	
)
