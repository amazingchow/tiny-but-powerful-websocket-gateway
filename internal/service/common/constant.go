package common

import (
	"context"
)

const (
	ReqHeaderKeyRequestId  string = "x-request-id"
	ReqHeaderKeyUid        string = "x-uid"
	ReqHeaderKeyAccount    string = "x-sec-account"
	ReqHeaderKeyToken      string = "x-sec-token"
	ReqHeaderKeyAppVersion string = "x-app-version"
)

type ContextKey string

const (
	ContextKeyTraceId ContextKey = "ctx-key-trace-id"
	ContextKeySpanId  ContextKey = "ctx-key-span-id"
)

const (
	LoggerKeyTraceId string = "trace-id"
	LoggerKeySpanId  string = "span-id"
	LoggerKeyUid     string = "uid"
)

func NewContextWithProvidedTraceIdAndSpanId(ctx context.Context, traceId, spanId string) context.Context {
	ctx = context.WithValue(ctx, ContextKeyTraceId, traceId)
	ctx = context.WithValue(ctx, ContextKeySpanId, spanId)
	return ctx
}

func TraceId(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyTraceId).(string); ok {
		return v
	}
	return ""
}

func SpanId(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeySpanId).(string); ok {
		return v
	}
	return ""
}
