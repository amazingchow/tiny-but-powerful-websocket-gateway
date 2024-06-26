package requestid

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/google/uuid"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
)

type ContextKey string

const (
	RequestIDKeyStr string     = "x-request-id"
	TraceIDKey      ContextKey = "x-trace-id"
	TraceIDKeyStr   string     = "x-trace-id"
	SpanIDKey       ContextKey = "x-span-id"
	SpanIDKeyStr    string     = "x-span-id"
)

type optionHandler func(ctx *context.Context)

func WithPrefix(prefix string) optionHandler {
	return func(ctx *context.Context) {
		traceId := (*ctx).Value(TraceIDKey).(string)
		if len(traceId) == 0 {
			*ctx = context.WithValue(*ctx, TraceIDKey, fmt.Sprintf("%s.%s", prefix, NewTraceIDKey()))
		} else {
			*ctx = context.WithValue(*ctx, TraceIDKey, fmt.Sprintf("%s.%s", prefix, traceId))
		}
		spanId := (*ctx).Value(SpanIDKey).(string)
		if len(spanId) == 0 {
			*ctx = context.WithValue(*ctx, SpanIDKey, fmt.Sprintf("%s.%s", prefix, NewTraceIDKey()))
		} else {
			*ctx = context.WithValue(*ctx, SpanIDKey, fmt.Sprintf("%s.%s", prefix, spanId))
		}
	}
}

func NewContext(option ...optionHandler) context.Context {
	ctx := context.WithValue(context.Background(), TraceIDKey, NewTraceIDKey())
	ctx = context.WithValue(ctx, SpanIDKey, NewSpanIDKey())
	for _, f := range option {
		f(&ctx)
	}
	return ctx
}

func NewContextWithProvidedTraceId(traceId string, option ...optionHandler) context.Context {
	ctx := context.WithValue(context.Background(), TraceIDKey, traceId)
	ctx = context.WithValue(ctx, SpanIDKey, NewSpanIDKey())
	for _, f := range option {
		f(&ctx)
	}
	return ctx
}

func NewContextWithProvidedTraceIdAndSpanId(traceId, spanId string, option ...optionHandler) context.Context {
	ctx := context.WithValue(context.Background(), TraceIDKey, traceId)
	ctx = context.WithValue(ctx, SpanIDKey, spanId)
	for _, f := range option {
		f(&ctx)
	}
	return ctx
}

func NewContextFromParent(ctxP context.Context, option ...optionHandler) context.Context {
	ctx := context.WithValue(ctxP, TraceIDKey, NewTraceIDKey())
	ctx = context.WithValue(ctx, SpanIDKey, NewSpanIDKey())
	for _, f := range option {
		f(&ctx)
	}
	return ctx
}

func NewContextFromParentWithProvidedTraceId(ctxP context.Context, traceId string, option ...optionHandler) context.Context {
	ctx := context.WithValue(ctxP, TraceIDKey, traceId)
	ctx = context.WithValue(ctx, SpanIDKey, NewSpanIDKey())
	for _, f := range option {
		f(&ctx)
	}
	return ctx
}

func NewContextFromParentWithProvidedTraceIdAndSpanId(ctxP context.Context, traceId, spanId string, option ...optionHandler) context.Context {
	ctx := context.WithValue(ctxP, TraceIDKey, traceId)
	ctx = context.WithValue(ctx, SpanIDKey, NewSpanIDKey())
	for _, f := range option {
		f(&ctx)
	}
	return ctx
}

func NewTraceIDKey() string {
	return uuid.New().String()
}

func TraceIDKeyFromContext(ctx context.Context) string {
	return ctx.Value(TraceIDKey).(string)
}

func NewSpanIDKey() string {
	return uuid.New().String()
}

func SpanIDKeyFromContext(ctx context.Context) string {
	return ctx.Value(SpanIDKey).(string)
}

func HandlePanic(ctx context.Context, f func(ctx context.Context)) {
	defer func() {
		if err := recover(); err != nil {
			stack := strings.Join(strings.Split(string(debug.Stack()), "\n")[2:], "\n")
			if len(stack) > 5000 {
				stack = stack[:5000]
			}
			logger.GetGlobalLogger().WithField(string(TraceIDKey), TraceIDKeyFromContext(ctx)).
				WithField(string(SpanIDKey), SpanIDKeyFromContext(ctx)).
				Errorf("panic, err:%v, stack:%v", err, stack)
		}
	}()
	f(ctx)
}
