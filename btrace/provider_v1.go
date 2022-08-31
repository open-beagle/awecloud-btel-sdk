package btrace

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	bresource "github.com/open-beagle/awecloud-btel-sdk/resource"
)

const (
	// OTel服务的访问地址
	otlp_endpoint = "BTEL_EXPORTER_OTLP_ENDPOINT"

	// 传播的自定义属性
	service_name = "BTEL_SERVICE_NAME"
)

// NewTracer , 新建一个追踪器
func NewTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	sService_Name := strings.TrimSpace(os.Getenv(service_name))
	if sService_Name == "" {
		return nil, nil
	}

	exporter, err := initTracerExporter(ctx)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(initTracerResource(ctx)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

func initTracerExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	sOTLP_Endpoint := strings.TrimSpace(os.Getenv(otlp_endpoint))

	if sOTLP_Endpoint == "" {
		exporter, err := stdout.New(stdout.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		return exporter, nil
	}

	sOTLP_Endpoint = strings.TrimPrefix(sOTLP_Endpoint, "http://")
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, sOTLP_Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Set up a trace exporter
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	if err != nil {
		// fmt.Errorf("creating OTLP trace exporter: %w", err)
		return nil, err
	}
	return exporter, nil
}

func initTracerResource(ctx context.Context) *resource.Resource {
	resources, _ := resource.New(ctx,
		resource.WithDetectors(bresource.FromEnv{}), // pull attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables
		resource.WithProcess(),                      // This option configures a set of Detectors that discover process information
	)
	return resources
}
