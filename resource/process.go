package resource

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"syscall"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	otelresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
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
	utsname := syscall.Utsname{}
	syscall.Uname(&utsname)
	description := charToString(utsname.Sysname) + " " + charToString(utsname.Release)
	return resource.NewWithAttributes(semconv.SchemaURL, attribute.String("os.description", description)), nil
}

type osType struct{}

func (osType) Detect(context.Context) (*otelresource.Resource, error) {
	return resource.NewWithAttributes(semconv.SchemaURL, attribute.String("os.type", runtime.GOOS)), nil
}

func charToString(c [65]int8) (str string) {
	for _, v := range c {
		if v == 0 {
			continue
		}
		str += fmt.Sprintf("%c", v)
	}
	return str
}
