package middleware

import (
	"net/http"

	log "github.com/totvs/go-sdk/log"
	adapter "github.com/totvs/go-sdk/log/adapter"
	tr "github.com/totvs/go-sdk/trace"
)

// MiddlewareOptions customizes the behavior of the HTTP middleware.
type MiddlewareOptions struct {
	// LogRequest controls whether the middleware emits a request-level log.
	LogRequest bool
	// InjectLogger controls whether the middleware stores the facade logger in the request context.
	InjectLogger bool
	// AddTraceHeader controls whether the middleware sets the trace header on the response.
	AddTraceHeader bool
}

// DefaultMiddlewareOptions are the defaults used by HTTPMiddlewareWithLogger.
var DefaultMiddlewareOptions = MiddlewareOptions{LogRequest: true, InjectLogger: true, AddTraceHeader: true}

// HTTPMiddlewareWithOptions returns a middleware using the provided base logger
// and the supplied options.
func HTTPMiddlewareWithOptions(base log.LoggerFacade, opts MiddlewareOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tid := r.Header.Get(tr.TraceIDHeader)
			if tid == "" {
				tid = r.Header.Get(tr.TraceIDCorrelationHeader)
			}
			if tid == "" {
				tid = tr.GenerateTraceID()
			}
			ctx := tr.ContextWithTrace(r.Context(), tid)

			// prepare a facade that includes trace
			l := base.WithTraceFromContext(ctx)
			// method/path are added as structured fields; trace_id is already
			// added by WithTraceFromContext above so avoid duplicating it here.
			l2 := l.WithFields(map[string]interface{}{"method": r.Method, "path": r.URL.Path})

			if opts.LogRequest {
				l2.Info().Msg("http request received")
				ctx = tr.ContextWithLogged(ctx)
			}
			if opts.InjectLogger {
				ctx = log.ContextWithLogger(ctx, l2)
			}
			if opts.AddTraceHeader {
				if w.Header().Get(tr.TraceIDHeader) == "" {
					w.Header().Set(tr.TraceIDHeader, tid)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// HTTPMiddlewareWithLogger is a convenience wrapper that uses default options.
func HTTPMiddlewareWithLogger(base log.LoggerFacade) func(http.Handler) http.Handler {
	return HTTPMiddlewareWithOptions(base, DefaultMiddlewareOptions)
}

// HTTPMiddleware is a convenience wrapper that uses the default facade logger.
func HTTPMiddleware(next http.Handler) http.Handler {
	// create a default adapter-backed logger at call time so callers can
	// influence behavior via environment variables (e.g., LOG_LEVEL) in tests.
	return HTTPMiddlewareWithLogger(adapter.NewDefaultLog())(next)
}

// GetLoggerFromRequest is a convenience helper for HTTP handlers.
// It returns a LoggerFacade extracted from the request context when present,
// otherwise returns the global logger. The second return value indicates
// whether the middleware already logged the request (so handlers can avoid
// duplicating the same request-level message).
func GetLoggerFromRequest(r *http.Request) (log.LoggerFacade, bool) {
	if r == nil {
		return log.GetGlobal(), false
	}
	if lf, ok := log.LoggerFromContext(r.Context()); ok {
		return lf, tr.LoggedFromContext(r.Context())
	}
	return log.GetGlobal(), false
}
