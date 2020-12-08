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

	"go.opentelemetry.io/otel"
	apitrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type contextKeyType int

// avoid use char `/` in string
const initialTraceIDAnnotationKey string = "trace.kubernetes.io.initial"

// avoid use char `/` in string
const spanContextAnnotationKey string = "trace.kubernetes.io.span.context"

const initialTraceIDBaggageKey label.Key = "Initial-Trace-Id"

// SpanContextWithObject returns a context.Context with Spacn and Baggage from the passed object
func SpanContextWithObject(ctx context.Context, meta metav1.Object) context.Context {
	return SpanContextFromAnnotations(ctx, meta.GetAnnotations())
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
	span := httpTraceSpan{
		spanContext: spanContext,
	}
	return apitrace.ContextWithSpan(ctx, span)
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
