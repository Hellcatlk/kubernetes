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
	"context"
	"time"

	apitrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
)

type httpTraceSpan struct {
	spanContext apitrace.SpanContext
}

// Tracer returns tracer used to create this span. Tracer cannot be nil.
func (span httpTraceSpan) Tracer() apitrace.Tracer {
	return nil
}

// End completes the span. No updates are allowed to span after it
// ends. The only exception is setting status of the span.
func (span httpTraceSpan) End(options ...apitrace.SpanOption) {
	return
}

// AddEvent adds an event to the span.
func (span httpTraceSpan) AddEvent(ctx context.Context, name string, attrs ...label.KeyValue) {
	return
}

// AddEventWithTimestamp adds an event with a custom timestamp
// to the span.
func (span httpTraceSpan) AddEventWithTimestamp(ctx context.Context, timestamp time.Time, name string, attrs ...label.KeyValue) {
	return
}

// IsRecording returns true if the span is active and recording events is enabled.
func (span httpTraceSpan) IsRecording() bool {
	return false
}

// RecordError records an error as a span event.
func (span httpTraceSpan) RecordError(ctx context.Context, err error, opts ...apitrace.ErrorOption) {
	return
}

// SpanContext returns span context of the span. Returned SpanContext is usable
// even after the span ends.
func (span httpTraceSpan) SpanContext() apitrace.SpanContext {
	return span.spanContext
}

// SetStatus sets the status of the span in the form of a code
// and a message.  SetStatus overrides the value of previous
// calls to SetStatus on the Span.
//
// The default span status is OK, so it is not necessary to
// explicitly set an OK status on successful Spans unless it
// is to add an OK message or to override a previous status on the Span.
func (span httpTraceSpan) SetStatus(code codes.Code, msg string) {
	return
}

// SetName sets the name of the span.
func (span httpTraceSpan) SetName(name string) {
	return
}

// SetAttributes set span attributes
func (span httpTraceSpan) SetAttributes(kv ...label.KeyValue) {
	return
}

// SetAttribute set singular span attribute, with type inference.
func (span httpTraceSpan) SetAttribute(k string, v interface{}) {
	return
}
