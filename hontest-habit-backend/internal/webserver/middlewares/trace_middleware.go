package middlewares

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/insanelyharsh/hontest-habit/internal/constants"
)

const traceIDHeader = "X-Request-Id"

// TraceID propagates an incoming X-Request-Id header, or generates one, and
// stores it on the request context and the response header.
func TraceID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(traceIDHeader)
		if traceID == "" {
			traceID = generateTraceID()
		}
		w.Header().Set(traceIDHeader, traceID)

		ctx := context.WithValue(r.Context(), constants.TraceIDContextKey, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TraceIDFromContext returns the trace ID set by TraceID, if any.
func TraceIDFromContext(ctx context.Context) string {
	traceID, _ := ctx.Value(constants.TraceIDContextKey).(string)
	return traceID
}

func generateTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
