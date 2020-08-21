package httptrace

import (
	"context"
	"net/http"

	"go.opencensus.io/trace"
)

type httpTraceHandler struct {
	Handler *http.Handler
}

func (h *httpTraceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get span context from context
	span := trace.FromContext(r.Context())
	ctx := SpanContextToContext(r.Context(), span.SpanContext())
	// get init trace id from request header
	var initTraceID string = ""
	if len(r.Header[initTraceIDHeaderKey]) != 0 {
		initTraceID = r.Header[initTraceIDHeaderKey][0]
	}
	ctx = context.WithValue(ctx, initTraceIDContextKey, initTraceID)
	// call next handler
	(*h.Handler).ServeHTTP(w, r.WithContext(ctx))
}
