package btrace

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	bresource "github.com/open-beagle/awecloud-btel-sdk/resource"
	"github.com/sethvargo/go-envconfig"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Tracer struct {
	BtelExporterEndpoint string `env:"BTEL_EXPORTER_OTLP_ENDPOINT"`
	BtelServiceName      string `env:"BTEL_SERVICE_NAME"`
	errorHandler         otel.ErrorHandler
	Resource             *resource.Resource
	stop                 []func()
	ctx                  context.Context
	log                  *zap.Logger
}

// IsValid check config and return error if config invalid
func (c *Tracer) isValid() error {
	if c.BtelExporterEndpoint == "" {
		return errors.New("empty BTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if c.BtelServiceName == "" {
		return errors.New("empty BTEL_SERVICE_NAME")
	}
	return nil
}

type Option func(*Tracer)

func New(opts ...Option) *Tracer {
	var c Tracer
	c.ctx = context.Background()
	// 日志
	c.setLogger()
	// 1. load env config
	envError := envconfig.Process(c.ctx, &c)
	if envError != nil {
		c.log.Error("btel", zap.Error(envError))
		return nil
	}

	// 2. load code config
	for _, opt := range opts {
		opt(&c)
	}

	// 3. merge resource
	// parseEnvKeys(&c)
	mergeResource(&c)
	if err := c.isValid(); err != nil {
		c.log.Error("btel", zap.Error(err))
		return nil
	}
	if err := start(&c); err != nil {
		c.log.Error("btel", zap.Error(err))
		return nil
	}
	return &c
}

func (t *Tracer) setLogger() {
	config := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	atom := zap.NewAtomicLevel()
	t.log = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		zapcore.Lock(os.Stdout),
		atom,
	)) // 根据上面的配置创建logger
	zap.ReplaceGlobals(t.log) // 替换zap库里全局的logger
	defer t.log.Sync()
	atom.UnmarshalText([]byte("debug"))
}

// Shutdown 优雅关闭，将OpenTelemetry SDK内存中的数据发送到服务端
func (t *Tracer) Shutdown() {
	for _, stop := range t.stop {
		stop()
	}
}

// 初始化Traces，默认全量上传
func (c *Tracer) initTracer(traceExporter trace.SpanExporter, stop func()) error {
	if traceExporter == nil {
		return nil
	}
	// 建议使用AlwaysSample全量上传Trace数据，若您的数据太多，可以使用sdktrace.ProbabilitySampler进行采样上传
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(
			traceExporter,
			sdktrace.WithMaxExportBatchSize(10),
		),
		sdktrace.WithResource(c.Resource),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	c.stop = append(c.stop, func() {
		tp.Shutdown(c.ctx)
		stop()
	})
	return nil
}

// 默认使用本机hostname作为hostname
func getDefaultResource(c *Tracer) *resource.Resource {
	hostname, _ := os.Hostname()
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(c.BtelServiceName),
		semconv.HostNameKey.String(hostname),
		semconv.ProcessPIDKey.Int(os.Getpid()),
		semconv.ProcessCommandKey.String(os.Args[0]),
	)
}

func mergeResource(c *Tracer) error {
	var e error
	if c.Resource, e = resource.Merge(getDefaultResource(c), c.Resource); e != nil {
		return e
	}

	resource.WithDetectors(bresource.FromEnv{})
	r := c.initTracerResource()
	if c.Resource, e = resource.Merge(c.Resource, r); e != nil {
		return e
	}

	newResource := resource.NewWithAttributes(semconv.SchemaURL)
	if c.Resource, e = resource.Merge(c.Resource, newResource); e != nil {
		return e
	}
	return nil
}

// Start 初始化OpenTelemetry SDK，需要把 ${endpoint} 替换为实际的地址
// 如果填写为stdout则为调试默认，数据将打印到标准输出
func start(c *Tracer) error {
	if c.errorHandler != nil {
		otel.SetErrorHandler(c.errorHandler)
	}
	traceExporter, traceExpStop, err := c.initOtelExporter(c.BtelExporterEndpoint, false)
	if err != nil {
		return err
	}
	err = c.initTracer(traceExporter, traceExpStop)
	if err != nil {
		return err
	}
	return err
}

// 初始化Exporter，如果otlpEndpoint传入的值为 stdout，则默认把信息打印到标准输出用于调试
func (c *Tracer) initOtelExporter(otlpEndpoint string, insecure1 bool) (trace.SpanExporter, func(), error) {
	var traceExporter trace.SpanExporter
	var err error

	var exporterStop = func() {
		if traceExporter != nil {
			traceExporter.Shutdown(c.ctx)
		}
	}

	if otlpEndpoint == "stdout" {
		// 使用Pretty的打印方式
		traceExporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, nil, err
		}
	} else if otlpEndpoint != "" {
		ctx, cancel := context.WithTimeout(c.ctx, time.Second)
		defer cancel()
		otlpEndpoint = strings.TrimPrefix(otlpEndpoint, "http://")
		conn, err := grpc.DialContext(ctx, otlpEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			return nil, nil, err
		}

		// Set up a trace exporter
		traceExporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			return nil, nil, err
		}
	}

	return traceExporter, exporterStop, nil
}

func (c *Tracer) initTracerResource() *resource.Resource {
	resources, _ := resource.New(c.ctx,
		resource.WithDetectors(bresource.FromEnv{}), // pull attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables
		resource.WithProcess(),                      // This option configures a set of Detectors that discover process information
		bresource.WithOtherProcess(),
	)
	return resources
}
