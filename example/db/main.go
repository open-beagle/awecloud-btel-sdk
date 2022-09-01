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
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gin-gonic/gin"
	"github.com/open-beagle/awecloud-btel-sdk/btrace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	if tracer := btrace.New(); tracer != nil {
		defer tracer.Shutdown()
	}
	db, err := connectDB()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	r := gin.Default()
	r.Use(otelgin.Middleware(os.Getenv("BTEL_SERVICE_NAME")))
	r.GET("/user/:name", func(c *gin.Context) {
		time, _ := query(c.Request.Context(), db)
		reply := fmt.Sprintf("CURRENT_TIMESTAMP: %s \n", time)
		c.String(200, reply)
	})
	r.Run(":8080")
}

func connectDB() (*sql.DB, error) {
	db, err := btrace.Open("mysql", "root:passwd123@tcp(k8s.wodcloud.com:33082)/trace?parseTime=true")
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	return db, err
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
