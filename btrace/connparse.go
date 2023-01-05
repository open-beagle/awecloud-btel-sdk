package btrace

import (
	"fmt"
	"strings"
)

func connParse(driver, conn string) (connection, user, dbName string) {
	fmt.Println(driver, conn)
	switch driver {
	case "postgres": //postgres://pqgotest:password@localhost/pqgotest?sslmode=verify-full port=5433 user=postgres password=123456 dbname=ficow sslmode=disable
		if strings.Contains(conn, "@") {
			arr := strings.Split(conn, "@")
			connection = strings.Split(arr[1], "/")[0]
			left := arr[0]
			if strings.Contains(left, "://") {
				left = strings.Split(left, "://")[1]
				user = strings.Split(left, ":")[0]
			}
			dbName = strings.Split(arr[1], "/")[1]
			if strings.Contains(dbName, "?") {
				dbName = strings.Split(dbName, "?")[0]
			}
		} else {
			arr := strings.Split(conn, " ")
			var host, port string
			for _, v := range arr {
				if strings.Contains(v, "=") {
					kv := strings.Split(v, "=")
					switch kv[0] {
					case "host":
						host = kv[1]
					case "port":
						port = kv[1]
					case "user":
						user = kv[1]
					case "dbname":
						dbName = kv[1]
					}
				}
			}
			connection = host + ":" + port
		}
	case "sqlite3":
	case "mysql": // root:password@tcp(mysql.istio-samples.svc.cluster.local:3306)/test root:password@(mysql.istio-samples:3306)/ysgz-ys?charset=utf8mb4
		arr := strings.Split(conn, "@")
		user = strings.Split(arr[0], ":")[0]
		connection = strings.Split(arr[1], "/")[0]
		if strings.HasPrefix(connection, "(") {
			connection = strings.ReplaceAll(connection, "(", "")
			connection = strings.ReplaceAll(connection, ")", "")
		}
		dbName = strings.Split(arr[1], "/")[1]
		if strings.Contains(dbName, "?") {
			dbName = strings.Split(dbName, "?")[0]
		}
	case "mssql":
	case "oracle":
	case "dameng":
	}
	return
}
