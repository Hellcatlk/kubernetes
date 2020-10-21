/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package httptrace

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"log"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/api/global"
	apitrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagators"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type contextKeyType int

// avoid use char `/` in string
const initialTraceIDAnnotationKey string = "trace.kubernetes.io.initial"

// avoid use char `/` in string
const spanContextAnnotationKey string = "trace.kubernetes.io.span.context"

var tracePropagator propagators.TraceContext
var baggagePropagator propagators.Baggage

const initialTraceIDBaggageKey label.Key = "Initial-Trace-Id"

// InitTracer ...
func InitTracer() func() {
	var err error
	exp, err := stdout.NewExporter(stdout.WithPrettyPrint())
	if err != nil {
		log.Panicf("failed to initialize stdout exporter %v\n", err)
		return nil
	}
	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithConfig(
			sdktrace.Config{
				DefaultSampler: sdktrace.NeverSample(),
			},
		),
		sdktrace.WithSpanProcessor(bsp),
	)
	global.SetTracerProvider(tp)
	return bsp.Shutdown
}

// WithTracingHandler inject span into http request context
func WithTracingHandler(handler http.Handler) http.Handler {
	return otelhttp.NewHandler(
		&httpTraceHandler{
			Handler: &handler,
		},
		"trace",
		otelhttp.WithPropagators(tracePropagator),
	)
}

// SpanContextToRequest inject span context in golang context to http request header
func SpanContextToRequest(ctx context.Context, req *http.Request) {
	spanContext := SpanContextFromContext(ctx)
	span := httpTraceSpan{
		spanContext: spanContext,
	}
	// inject span context into request header
	tracePropagator.Inject(apitrace.ContextWithSpan(context.Background(), span), req.Header)
	// inject init trace id into request header
	baggagePropagator.Inject(ctx, req.Header)
}

// SpanContextFromRequestHeader extract span context from http request header
func SpanContextFromRequestHeader(req *http.Request) context.Context {
	// get span context from request header
	ctx := tracePropagator.Extract(req.Context(), req.Header)
	// get init trace id from request header
	return baggagePropagator.Extract(ctx, req.Header)
}

// SpanContextFromRequestContext extract span context from http request context
func SpanContextFromRequestContext(req *http.Request) context.Context {
	// get span context from span
	spanContext := SpanFromContext(req.Context()).SpanContext()
	// inject span context into golang context
	ctx := SpanContextToContext(req.Context(), spanContext)
	// get init trace id from request header
	return baggagePropagator.Extract(ctx, req.Header)
}

// SpanContextFromAnnotations get span context from annotations
func SpanContextFromAnnotations(ctx context.Context, annotations map[string]string) context.Context {
	// get init trace id from annotations
	ctx = otel.ContextWithBaggageValues(
		ctx,
		label.KeyValue{
			Key:   initialTraceIDBaggageKey,
			Value: label.StringValue(annotations[initialTraceIDAnnotationKey]),
		},
	)
	// get span context from annotations
	spanContext, err := decodeSpanContext(annotations[spanContextAnnotationKey])
	if err != nil {
		return ctx
	}
	return SpanContextToContext(ctx, spanContext)
}

// decodeSpanContext decode encodedSpanContext to spanContext
func decodeSpanContext(encodedSpanContext string) (apitrace.SpanContext, error) {
	// decode to byte
	byteList := make([]byte, base64.StdEncoding.DecodedLen(len(encodedSpanContext)))
	l, err := base64.StdEncoding.Decode(byteList, []byte(encodedSpanContext))
	if err != nil {
		return apitrace.EmptySpanContext(), err
	}
	byteList = byteList[:l]
	// decode to span context
	buffer := bytes.NewBuffer(byteList)
	spanContext := apitrace.SpanContext{}
	err = binary.Read(buffer, binary.LittleEndian, &spanContext)
	if err != nil {
		return apitrace.EmptySpanContext(), err
	}
	return spanContext, nil
}

// SpanFromContext get span from golang context
func SpanFromContext(ctx context.Context) apitrace.Span {
	return apitrace.SpanFromContext(ctx)
}

// SpanContextToContext set span context to golang context
func SpanContextToContext(ctx context.Context, spanContext apitrace.SpanContext) context.Context {
	return apitrace.ContextWithRemoteSpanContext(ctx, spanContext)
}

// SpanContextFromContext get span context from golang context
func SpanContextFromContext(ctx context.Context) apitrace.SpanContext {
	return apitrace.RemoteSpanContextFromContext(ctx)
}
