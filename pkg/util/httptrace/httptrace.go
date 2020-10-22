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
	"fmt"
	"reflect"

	"go.opentelemetry.io/otel"
	apitrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

type contextKeyType int

// avoid use char `/` in string
const initialTraceIDAnnotationKey string = "trace.kubernetes.io.initial"

// avoid use char `/` in string
const spanContextAnnotationKey string = "trace.kubernetes.io.span.context"

const initialTraceIDBaggageKey label.Key = "Initial-Trace-Id"

// SpanContextFromAnnotations extera span context from annotations
func SpanContextFromAnnotations(ctx context.Context, annotations map[string]string) context.Context {
	// get init trace id from annotations
	ctx = otel.ContextWithBaggageValues(
		ctx,
		label.KeyValue{
			Key:   initialTraceIDBaggageKey,
			Value: label.StringValue(annotations[initialTraceIDAnnotationKey]),
		},
	)
	spanContext, _ := decodeSpanContext(annotations[spanContextAnnotationKey])
	// get span context from annotations
	return SpanContextToContext(ctx, spanContext)
}

// SpanContextToAnnotations put inject span context into annotations
func SpanContextToAnnotations(ctx context.Context, annotations *map[string]string) {
	if *annotations == nil {
		*annotations = make(map[string]string)
	}
	// get init trace id from ctx, inject into annotations
	if otel.BaggageValue(ctx, initialTraceIDBaggageKey).AsString() != "" {
		(*annotations)[initialTraceIDAnnotationKey] = otel.BaggageValue(ctx, initialTraceIDBaggageKey).AsString()
	}
	// get spancontext from ctx, inject into annotations
	spanContext := SpanContextFromContext(ctx)
	if !reflect.DeepEqual(spanContext, apitrace.EmptySpanContext()) {
		encodeSpanContext, _ := encodedSpanContext(spanContext)
		(*annotations)[spanContextAnnotationKey] = encodeSpanContext
	}
}

// SpanContextToContext set span context to golang context
func SpanContextToContext(ctx context.Context, spanContext apitrace.SpanContext) context.Context {
	return apitrace.ContextWithRemoteSpanContext(ctx, spanContext)
}

// SpanContextFromContext get span context from golang context
func SpanContextFromContext(ctx context.Context) apitrace.SpanContext {
	return apitrace.RemoteSpanContextFromContext(ctx)
}

// encodedSpanContext encode span to string
func encodedSpanContext(spanContext apitrace.SpanContext) (string, error) {
	if reflect.DeepEqual(spanContext, apitrace.SpanContext{}) {
		return "", fmt.Errorf("span context is nil")
	}
	// encode to byte
	buffer := new(bytes.Buffer)
	err := binary.Write(buffer, binary.LittleEndian, spanContext)
	if err != nil {
		return "", err
	}
	// encode to string
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
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
