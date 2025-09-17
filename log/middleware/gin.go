package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/totvs/go-sdk/log"
	adapter "github.com/totvs/go-sdk/log/adapter"
	tr "github.com/totvs/go-sdk/trace"
)

// GinMiddlewareWithOptions returns a gin.HandlerFunc that integrates the
// LoggerFacade with Gin. It follows the same options used by the HTTP
// middleware so callers can reuse the same configuration semantics.
func GinMiddlewareWithOptions(base log.LoggerFacade, opts MiddlewareOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		tid := c.GetHeader(tr.TraceIDHeader)
		if tid == "" {
			tid = c.GetHeader(tr.TraceIDCorrelationHeader)
		}
		if tid == "" {
			tid = tr.GenerateTraceID()
		}

		ctx := tr.ContextWithTrace(c.Request.Context(), tid)

		// prepare a facade that includes trace
		l := base.WithTraceFromContext(ctx)
		l2 := l.WithFields(map[string]interface{}{"method": c.Request.Method, "path": c.Request.URL.Path})

		if opts.LogRequest {
			l2.Info().Msg("http request received")
			ctx = tr.ContextWithLogged(ctx)
		}
		if opts.InjectLogger {
			ctx = log.ContextWithLogger(ctx, l2)
			c.Request = c.Request.WithContext(ctx)
		} else {
			// still attach trace ctx so downstream code can read trace id
			c.Request = c.Request.WithContext(ctx)
		}
		if opts.AddTraceHeader {
			if c.Writer.Header().Get(tr.TraceIDHeader) == "" {
				c.Writer.Header().Set(tr.TraceIDHeader, tid)
			}
		}

		c.Next()

		// after handler: emit completion log with status and latency
		latency := time.Since(start)
		status := c.Writer.Status()
		size := c.Writer.Size()
		l2.WithFields(map[string]interface{}{"status": status, "latency_ms": latency.Milliseconds(), "size": size}).Info().Msg("http request completed")
	}
}

// GinMiddlewareWithLogger is a convenience wrapper that uses default options.
func GinMiddlewareWithLogger(base log.LoggerFacade) gin.HandlerFunc {
	return GinMiddlewareWithOptions(base, DefaultMiddlewareOptions)
}

// GinMiddleware is a convenience wrapper that uses the default facade logger.
func GinMiddleware() gin.HandlerFunc {
	// create a default impl-backed logger at call time so callers can
	// influence behavior via environment variables (e.g., LOG_LEVEL) in tests.
	return GinMiddlewareWithLogger(adapter.NewDefaultLog())
}

// GetLoggerFromGinContext extracts a LoggerFacade from a gin.Context. The
// boolean indicates whether the middleware already logged the request.
func GetLoggerFromGinContext(c *gin.Context) (log.LoggerFacade, bool) {
	if c == nil {
		return log.GetGlobal(), false
	}
	if lf, ok := log.LoggerFromContext(c.Request.Context()); ok {
		return lf, tr.LoggedFromContext(c.Request.Context())
	}
	return log.GetGlobal(), false
}
