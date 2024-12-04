package utils

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type attrKey string

const (
	attrkey attrKey = "attrkv"
)

func Span(
	ctx context.Context, spanName string, opts ...trace.SpanStartOption) (spanctx context.Context, span trace.Span) {
	tr := otel.Tracer("gitea-http-client-tracer")
	spanctx, span = tr.Start(ctx, spanName)
	// 设置 Attr
	attrkv, ok := ctx.Value(attrkey).(map[string]string)
	if ok {
		SpanSetStringAttr(span, attrkv)
	}

	return spanctx, span
}

func SpanSetStringAttr(span trace.Span, kvs map[string]string) {
	attrkv := []attribute.KeyValue{}

	for k, v := range kvs {
		attrkv = append(attrkv, attribute.KeyValue{
			Key:   attribute.Key(k),
			Value: attribute.StringValue(v),
		})
	}

	span.SetAttributes(attrkv...)
}

func SpanContextWithAttr(ctx context.Context, kv map[string]string) context.Context {

	value := ctx.Value(attrkey)
	attrkv, ok := value.(map[string]string)
	if !ok {
		attrkv = make(map[string]string, 0)
	}

	for k, v := range kv {
		attrkv[k] = v
	}

	return context.WithValue(ctx, attrkey, attrkv)
}
