// Copyright The AliyunSLS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/attribute"

	"github.com/gorilla/mux"
	"github.com/open-beagle/awecloud-btel-sdk/btrace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel/trace"
)

func main() {

	slsConfig, err := btrace.NewConfig()
	// 如果初始化失败则panic，可以替换为其他错误处理方式
	if err != nil {
		panic(err)
	}
	if err := btrace.Start(slsConfig); err != nil {
		panic(err)
	}
	defer btrace.Shutdown(slsConfig)

	r := mux.NewRouter()
	r.Use(otelmux.Middleware("my-server"))
	r.HandleFunc("/users/{id:[0-9]+}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		name := getUser(r.Context(), id)
		reply := fmt.Sprintf("user %s (id %s)\n", name, id)
		_, _ = w.Write(([]byte)(reply))
	}))
	http.Handle("/", r)
	fmt.Println("Now listen port 8080, you can visit 127.0.0.1:8080/users/xxx .")
	_ = http.ListenAndServe(":8080", nil)
}

func getUser(ctx context.Context, id string) string {
	if id == "123" {
		return "otelmux tester"
	}
	// 如果需要记录一些事件，可以获取Context中的span并添加Event（非必要步骤）
	span := trace.SpanFromContext(ctx)
	span.AddEvent("unknown user id : "+id, trace.WithAttributes(attribute.KeyValue{
		Key:   "label-key-1",
		Value: attribute.StringValue("label-value-1"),
	}))
	return "unknown"
}
