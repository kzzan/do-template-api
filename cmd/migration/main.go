package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"github.com/jmoiron/sqlx"

	"do-template-api/pkg/config"
)

// migration 执行 storage/schema.sql（按分号切分语句逐条执行）。
// ponytail: 简易迁移，无版本管理；正式项目需要演进式迁移时换 golang-migrate / goose
func main() {
	confPath := flag.String("conf", "config/local.yml", "config path")
	schemaPath := flag.String("schema", "storage/schema.sql", "schema file path")
	flag.Parse()

	conf := config.NewConfig(*confPath)
	driver, dsn := conf.GetString("data.db.driver"), conf.GetString("data.db.dsn")

	db, err := sqlx.Connect(driver, dsn)
	if err != nil {
		log.Fatalf("connect %s: %v", driver, err)
	}
	defer db.Close()

	b, err := os.ReadFile(*schemaPath)
	if err != nil {
		log.Fatalf("read schema: %v", err)
	}
	sqlText := strings.TrimSpace(string(b))

	for _, stmt := range strings.Split(sqlText, ";") {
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("exec %q: %v", strings.TrimSpace(stmt), err)
		}
	}
	fmt.Println("migration done:", *schemaPath)
}
