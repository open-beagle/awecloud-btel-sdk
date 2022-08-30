package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/open-beagle/awecloud-btel-sdk/btrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	config, err := btrace.NewConfig()
	// 使用panic()，表示如果初始化失败则程序直接异常退出，您也可以使用其他错误处理方式。
	if err != nil {
		panic(err)
	}
	if err := btrace.Start(config); err != nil {
		panic(err)
	}
	defer btrace.Shutdown(config)

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
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
