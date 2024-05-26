package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/178inaba/today-duty-bot/handler"
	"github.com/178inaba/today-duty-bot/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/slack-go/slack"
)

func main() {
	ctx := context.Background()

	db, err := openSqlxDB(
		os.Getenv("MYSQL_USER"),
		os.Getenv("MYSQL_PASSWORD"),
		os.Getenv("MYSQL_PROTOCOL"),
		os.Getenv("MYSQL_ADDRESS"),
		os.Getenv("MYSQL_DB_NAME"),
	)
	if err != nil {
		log.Fatalf("Open database: %v.", err)
	}
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Ping database: %v.", err)
	}

	memberRepo := repository.NewMemberRepository(db)
	dutyHistoryRepo := repository.NewDutyHistoryRepository(db)

	h := handler.NewHandler(
		memberRepo,
		dutyHistoryRepo,
		slack.New(os.Getenv("SLACK_TOKEN")),
		os.Getenv("SLACK_SIGNING_SECRET"),
	)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/duties", h.CreateDuty)
	r.Post("/events", h.ReceiveEvent)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on port %s.", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("End listen and serve: %v.", err)
	}
}

func openSqlxDB(user, passwd, net, addr, dbName string) (*sqlx.DB, error) {
	c := mysql.NewConfig()
	c.User = user
	c.Passwd = passwd
	c.Net = net
	c.Addr = addr
	c.DBName = dbName
	c.Collation = "utf8mb4_bin"
	c.ParseTime = true

	db, err := sqlx.Open("mysql", c.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open sqlx: %w", err)
	}

	return db, nil
}
