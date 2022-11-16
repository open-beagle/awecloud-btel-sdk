package btrace

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"xorm.io/xorm"
	"xorm.io/xorm/contexts"
)

const (
	// skywalking 没有 xorm 对应的 component id, 先复用下 Mysql
	// https://github.com/apache/skywalking/blob/42c8cebbc1bb30b003db477b86ec8f7360a1e1aa/oap-server/server-bootstrap/src/main/resources/component-libraries.yml#L47
	ComponentIDMysql  int32 = 5
	ComponentIDGoXorm int32 = 5008
)

type oteltraceHook struct {
	tracer oteltrace.Tracer
	engine *xorm.Engine
}

func NewTraceHook(engine *xorm.Engine, tracer oteltrace.Tracer) *oteltraceHook {
	return &oteltraceHook{tracer: tracer, engine: engine}
}

func WrapEngine(e *xorm.Engine, tracer oteltrace.Tracer) {
	e.AddHook(NewTraceHook(e, tracer))
}

func (h *oteltraceHook) BeforeProcess(c *contexts.ContextHook) (context.Context, error) {
	commonAttrs := []attribute.KeyValue{
		attribute.String("db.statement", c.SQL),
		attribute.String("db.connection_string", h.engine.DataSourceName()),
		attribute.String("db.name", h.engine.Dialect().URI().DBName),
		attribute.String("db.system", string(h.engine.Dialect().URI().DBType)),
		attribute.String("db.user", h.engine.Dialect().URI().User),
	}
	_, iSpan := h.tracer.Start(c.Ctx, c.SQL, trace.WithAttributes(commonAttrs...))
	ctx := context.WithValue(c.Ctx, "xorm span", iSpan)
	return ctx, nil
}

func (h *oteltraceHook) AfterProcess(c *contexts.ContextHook) error {
	span := c.Ctx.Value("xorm span").(oteltrace.Span)
	defer span.End()
	return nil
}
