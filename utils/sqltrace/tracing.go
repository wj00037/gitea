package sqltrace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.opentelemetry.io/otel/trace"
	"xorm.io/xorm/contexts"
)

const (
	tracerName = "gitea_xorm_tracer"
)

type OtelHook struct {
	config *traceConfig
}

func Hook(attrSetterArray ...AttrSetter) contexts.Hook {
	cfg := &traceConfig{}
	for _, attrSetter := range attrSetterArray {
		attrSetter.set(cfg)
	}
	cfg.tracer = otel.Tracer(tracerName)
	for _, attr := range cfg.attrs {
		if attr.Key == semconv.DBNameKey {
			cfg.db = attr.Value.AsString()
		}
	}
	return &OtelHook{
		config: cfg,
	}
}

func (hook *OtelHook) BeforeProcess(ctxHook *contexts.ContextHook) (context.Context, error) {
	spanName := "gitea-db"
	if len(hook.config.db) != 0 {
		spanName = hook.config.db
	}
	ctx, _ := hook.config.tracer.Start(ctxHook.Ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindClient),
	)
	return ctx, nil
}

func (hook *OtelHook) AfterProcess(ctxHook *contexts.ContextHook) error {
	span := trace.SpanFromContext(ctxHook.Ctx)
	attrs := make([]attribute.KeyValue, 0)
	defer span.End()

	attrs = append(attrs, hook.config.attrs...)
	attrs = append(attrs, attribute.Key("db.orm").String("xorm"))
	attrs = append(attrs, semconv.DBStatement(replaceSql(ctxHook.SQL, ctxHook.Args)))

	if ctxHook.Err != nil {
		span.RecordError(ctxHook.Err)
		span.SetStatus(codes.Error, ctxHook.Err.Error())
	}
	span.SetAttributes(attrs...)
	return nil
}
