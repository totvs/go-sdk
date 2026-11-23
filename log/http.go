package log

import (
	"net"
	"net/http"
	"strings"
)

// Structured field names used by LoggerFacade.WithHTTPRequest,
// LoggerFacade.WithHTTPLatency and HTTPFields.
const (
	HTTPClientIPField           = "http.client.ip"
	HTTPClientUserAgentField    = "http.client.user-agent"
	HTTPRequestMethodField      = "http.request.method"
	HTTPRequestPathField        = "http.request.path"
	HTTPResponseStatusCodeField = "http.response.status.code"
	HTTPResponseBodySizeField   = "http.response.body.size"
	HTTPResponseLatencyField    = "http.response.latency"
)

// HTTPFields builds the structured HTTP fields from the native request and
// response. Status code and body size come from resp.StatusCode and
// resp.ContentLength (-1 when unknown). A nil request or response omits the
// corresponding fields. Latency is not included; use
// LoggerFacade.WithHTTPLatency. It is the shared field builder used by
// LoggerFacade.WithHTTPRequest implementations.
func HTTPFields(r *http.Request, resp *http.Response) map[string]any {
	fields := map[string]any{}
	if resp != nil {
		fields[HTTPResponseStatusCodeField] = resp.StatusCode
		fields[HTTPResponseBodySizeField] = resp.ContentLength
	}
	if r == nil {
		return fields
	}
	fields[HTTPClientIPField] = ClientIP(r)
	fields[HTTPClientUserAgentField] = r.UserAgent()
	fields[HTTPRequestMethodField] = r.Method
	if r.URL != nil {
		fields[HTTPRequestPathField] = r.URL.Path
	}
	return fields
}

// ClientIP returns the client IP for the request. It checks, in order, the
// first entry of X-Forwarded-For, X-Real-Ip and finally RemoteAddr.
//
// NOTE: X-Forwarded-For and X-Real-Ip are client-controlled unless a trusted
// proxy overwrites them; use the value for logging only, never for
// authorization decisions.
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		if ip := strings.TrimSpace(first); ip != "" {
			return ip
		}
	}
	if xr := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xr != "" {
		return xr
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
