package resource

import (
	"context"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	otelresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"golang.org/x/sys/unix"
)

func WithOtherProcess() resource.Option {
	return resource.WithDetectors(
		telemetrySdkVersion{},
		telemetrySdkLanguage{},
		osDescription{},
		osType{},
	)
}

type telemetrySdkVersion struct{}

// Detect collects resources from environment.
func (telemetrySdkVersion) Detect(context.Context) (*otelresource.Resource, error) {
	version := runtime.Version()
	if strings.Contains(version, "go") {
		version = strings.Replace(version, "go", "", -1)
	}
	return resource.NewWithAttributes(semconv.SchemaURL, attribute.String("telemetry.sdk.version", version)), nil
}

type telemetrySdkLanguage struct{}

// Detect collects resources from environment.
func (telemetrySdkLanguage) Detect(context.Context) (*otelresource.Resource, error) {
	return resource.NewWithAttributes(semconv.SchemaURL, attribute.String("telemetry.sdk.language", "go")), nil
}

type osDescription struct{}

func (osDescription) Detect(context.Context) (*otelresource.Resource, error) {
	utsname := unix.Utsname{}
	unix.Uname(&utsname)
	description := charToString(utsname.Sysname[:]) + " " + charToString(utsname.Release[:])
	return resource.NewWithAttributes(semconv.SchemaURL, attribute.String("os.description", description)), nil
}

type osType struct{}

func (osType) Detect(context.Context) (*otelresource.Resource, error) {
	return resource.NewWithAttributes(semconv.SchemaURL, attribute.String("os.type", runtime.GOOS)), nil
}

func charToString(arr []byte) string {
	b := make([]byte, 0, len(arr))
	for _, v := range arr {
		if v == 0x00 {
			break
		}
		b = append(b, byte(v))
	}
	return string(b)
}
