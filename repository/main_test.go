package repository

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var testDB *sqlx.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Open database.
	c := mysql.NewConfig()
	c.User = os.Getenv("MYSQL_USER")
	c.Passwd = os.Getenv("MYSQL_PASSWORD")
	c.Net = os.Getenv("MYSQL_PROTOCOL")
	c.Addr = os.Getenv("MYSQL_ADDRESS")
	c.DBName = os.Getenv("MYSQL_DB_NAME")
	c.Collation = "utf8mb4_bin"
	c.ParseTime = true
	db, err := sqlx.Open("mysql", c.FormatDSN())
	if err != nil {
		log.Fatalf("Open database: %v.", err)
	}
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Ping database: %v.", err)
	}
	testDB = db

	os.Exit(m.Run())
}
