package httptrace

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

var defaultFormat propagation.HTTPFormat = &b3.HTTPFormat{}

type contextKeyType int

const initTraceIDContextKey contextKeyType = 0
const traceContextKey contextKeyType = 1

const initTraceIDHeaderKey string = "Init-Traceid"
const initTraceIDAnnotationKey string = "trace.kubernetes.io.init"
const traceAnnotationKey string = "trace.kubernetes.io.context"

// InitializeExporter takes a ServiceType and sets the global OpenCensus exporter
// to export to that service on a specified Zipkin instance
func InitializeExporter(service string) {
	// create ocagent exporter
	exp, err := ocagent.NewExporter(ocagent.WithInsecure(), ocagent.WithServiceName(string(service)))
	if err != nil {
		log.Fatalf("Failed to create the agent exporter: %v", err)
	}
	// Only sample when the propagated parent SpanContext is sampled
	// Use ProbabilitySampler because it propagates the parent sampling decision.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(0)})
	trace.RegisterExporter(exp)
	return
}

// WithTracingHandler inject span context into http request
func WithTracingHandler(handler http.Handler) http.Handler {
	return &ochttp.Handler{
		Handler: &httpTraceHandler{
			Handler: &handler,
		},
		StartOptions: trace.StartOptions{Sampler: trace.ProbabilitySampler(0)},
	}
}

// SpanContextToRequest put span context to http request
func SpanContextToRequest(ctx context.Context, req *http.Request) {
	// inject span context into request header
	spanContext := SpanContextFromContext(ctx)
	defaultFormat.SpanContextToRequest(spanContext, req)
	// inject init trace id into request header
	initTraceID := ctx.Value(initTraceIDContextKey)
	if initTraceID != nil && initTraceID.(string) != "" {
		req.Header.Set(initTraceIDHeaderKey, initTraceID.(string))
	}
}

// SpanContextFromRequestHeader get span context from http request header
func SpanContextFromRequestHeader(ctx context.Context, req *http.Request) context.Context {
	spanContext, ok := defaultFormat.SpanContextFromRequest(req)
	if !ok {
		return ctx
	}
	ctx = SpanContextToContext(ctx, spanContext)
	return context.WithValue(ctx, initTraceIDContextKey, req.Header.Get(initTraceIDHeaderKey))
}

// SpanContextFromRequestContext get span context from http request context
func SpanContextFromRequestContext(ctx context.Context, req *http.Request) context.Context {
	span := trace.FromContext(req.Context())
	if span == nil {
		return ctx
	}
	ctx = SpanContextToContext(ctx, span.SpanContext())
	var initTraceID string = ""
	if len(req.Header[initTraceIDHeaderKey]) != 0 {
		initTraceID = req.Header[initTraceIDHeaderKey][0]
	}
	return context.WithValue(ctx, initTraceIDContextKey, initTraceID)
}

// SpanContextFromAnnotations get span context from annotations
func SpanContextFromAnnotations(annotations map[string]string) context.Context {
	// get init trace id from annotations
	ctx := context.WithValue(context.Background(), initTraceIDContextKey, annotations[initTraceIDAnnotationKey])
	// get init span context from annotations
	spanContext, err := decodeSpanContext(annotations[traceAnnotationKey])
	if err != nil {
		return ctx
	}
	return SpanContextToContext(ctx, spanContext)
}

// decodeSpanContext decode span context
func decodeSpanContext(encodedSpanContext string) (trace.SpanContext, error) {
	rawContextBytes := make([]byte, base64.StdEncoding.DecodedLen(len(encodedSpanContext)))
	l, err := base64.StdEncoding.Decode(rawContextBytes, []byte(encodedSpanContext))
	if err != nil {
		return trace.SpanContext{}, err
	}
	rawContextBytes = rawContextBytes[:l]
	spanContext, ok := propagation.FromBinary(rawContextBytes)
	if !ok {
		return trace.SpanContext{}, fmt.Errorf("has an unsupported version ID or contains no TraceID")
	}
	return spanContext, nil
}

// SpanContextFromContext get span context from golang context
func SpanContextFromContext(ctx context.Context) trace.SpanContext {
	spanContext := ctx.Value(traceContextKey)
	if spanContext == nil {
		return trace.SpanContext{}
	}
	return spanContext.(trace.SpanContext)
}

// SpanContextToContext put span context to golang context
func SpanContextToContext(ctx context.Context, spanContext trace.SpanContext) context.Context {
	return context.WithValue(ctx, traceContextKey, spanContext)
}
