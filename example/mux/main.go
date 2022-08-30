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
	"database/sql"
	"fmt"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"

	"github.com/XSAM/otelsql"
	"github.com/gorilla/mux"
	"github.com/open-beagle/awecloud-btel-sdk/btrace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

var mysqlDSN = "root:passwd123@tcp(k8s.wodcloud.com:33082)/trace?parseTime=true"

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
	r.HandleFunc("/current", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time := queryDb(r.Context())
		reply := fmt.Sprintf("CURRENT_TIMESTAMP: %s \n", time)
		_, _ = w.Write(([]byte)(reply))
	}))
	http.Handle("/", r)
	fmt.Println("Now listen port 8080, you can visit 127.0.0.1:8080/users/xxx .")
	_ = http.ListenAndServe(":8080", nil)
}

func queryDb(ctx context.Context) string {
	db, err := otelsql.Open("mysql", mysqlDSN, otelsql.WithAttributes(
		semconv.DBSystemMySQL,
	))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
		semconv.DBSystemMySQL,
	))
	if err != nil {
		panic(err)
	}
	t, _ := query(ctx, db)
	return t.String()
}

func query(ctx context.Context, db *sql.DB) (t time.Time, err error) {
	// Make a query
	rows, err := db.QueryContext(ctx, `SELECT CURRENT_TIMESTAMP`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&t)
		if err != nil {
			return
		}
	}
	return
}
