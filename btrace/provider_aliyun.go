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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	BtelExporterEndpoint string `env:"BTEL_EXPORTER_OTLP_ENDPOINT,default=stdout"`
	BtelServiceName      string `env:"BTEL_SERVICE_NAME"`
	errorHandler         otel.ErrorHandler
	Resource             *resource.Resource
	stop                 []func()
	ctx                  context.Context
}

// IsValid check config and return error if config invalid
func (c *Config) IsValid() error {
	if c.BtelExporterEndpoint == "" {
		return errors.New("empty btel exporter endpoint")
	}
	if c.BtelServiceName == "" {
		return errors.New("empty btel service name")
	}
	return nil
}

// WithTraceExporterEndpoint configures the endpoint for sending traces via OTLP
// 配置Trace的输出地址，如果配置为空则禁用Trace功能，配置为stdout则打印到标准输出用于测试
func WithTraceExporterEndpoint(url string) Option {
	return func(c *Config) {
		c.BtelExporterEndpoint = url
	}
}

// WithServiceName configures a "service.name" resource label
// 配置服务名称
func WithServiceName(name string) Option {
	return func(c *Config) {
		c.BtelServiceName = name
	}
}

type Option func(*Config)

func NewConfig(opts ...Option) (*Config, error) {
	var c Config
	c.ctx = context.Background()

	// 1. load env config
	envError := envconfig.Process(c.ctx, &c)
	if envError != nil {
		return nil, envError
	}

	// 2. load code config
	for _, opt := range opts {
		opt(&c)
	}

	// 3. merge resource
	// parseEnvKeys(&c)
	mergeResource(&c)
	return &c, c.IsValid()
}

// 初始化Traces，默认全量上传
func (c *Config) initTracer(traceExporter trace.SpanExporter, stop func(), config *Config) error {
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
		tp.Shutdown(context.Background())
		stop()
	})
	return nil
}

// 默认使用本机hostname作为hostname
func getDefaultResource(c *Config) *resource.Resource {
	hostname, _ := os.Hostname()
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(c.BtelServiceName),
		semconv.HostNameKey.String(hostname),
		semconv.ProcessPIDKey.Int(os.Getpid()),
		semconv.ProcessCommandKey.String(os.Args[0]),
	)
}

func mergeResource(c *Config) error {
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
func Start(c *Config) error {
	if c.errorHandler != nil {
		otel.SetErrorHandler(c.errorHandler)
	}
	traceExporter, traceExpStop, err := c.initOtelExporter(c.BtelExporterEndpoint, false)
	if err != nil {
		return err
	}
	err = c.initTracer(traceExporter, traceExpStop, c)
	if err != nil {
		return err
	}
	return err
}

// Shutdown 优雅关闭，将OpenTelemetry SDK内存中的数据发送到服务端
func Shutdown(c *Config) {
	for _, stop := range c.stop {
		stop()
	}
}

// 初始化Exporter，如果otlpEndpoint传入的值为 stdout，则默认把信息打印到标准输出用于调试
func (c *Config) initOtelExporter(otlpEndpoint string, insecure1 bool) (trace.SpanExporter, func(), error) {
	var traceExporter trace.SpanExporter
	var err error

	var exporterStop = func() {
		if traceExporter != nil {
			traceExporter.Shutdown(context.Background())
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

func (c *Config) initTracerResource() *resource.Resource {
	resources, _ := resource.New(c.ctx,
		resource.WithDetectors(bresource.FromEnv{}), // pull attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables
		resource.WithProcess(),                      // This option configures a set of Detectors that discover process information
	)
	return resources
}
