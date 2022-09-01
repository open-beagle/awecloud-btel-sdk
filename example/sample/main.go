package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/open-beagle/awecloud-btel-sdk/btrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap/zapcore"
)

func main() {
	if tracer := btrace.New(); tracer != nil {
		defer tracer.Shutdown()
	}

	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		if time.Now().Unix()%10 == 0 {
			_, _ = io.WriteString(w, "Hello, world!\n")
		} else {
			// 如果您需要记录一些事件，可以获取Context中的Span并添加Event。
			// ctx := req.Context()
			// span := trace.SpanFromContext(ctx)
			// span.AddEvent("say : Hello, I am david", trace.WithAttributes(label.KeyValue{
			// 	Key:   "label-key-1",
			// 	Value: label.StringValue("label-value-1"),
			// }))
			_, _ = io.WriteString(w, "Hello, I am david!\n")
		}
	}

	// 使用otel net/http的自动注入方式，只需要使用otelhttp.NewHandler包裹http.Handler即可。
	otelHandler := otelhttp.NewHandler(http.HandlerFunc(helloHandler), "Hello")

	http.Handle("/hello", otelHandler)
	fmt.Println("Now listen port 8080, you can visit 127.0.0.1:8080/hello .")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

// init log config
func initLogConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
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
}
