package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
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

	h := NewHandler(
		slack.New(os.Getenv("SLACK_TOKEN")),
		os.Getenv("SLACK_SIGNING_SECRET"),
	)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/events-endpoint", h.HelloWorld)

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

type Handler struct {
	slackClient        *slack.Client
	slackSigningSecret string
}

func NewHandler(slackClient *slack.Client, slackSigningSecret string) *Handler {
	return &Handler{
		slackClient:        slackClient,
		slackSigningSecret: slackSigningSecret,
	}
}

func (h *Handler) HelloWorld(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sv, err := slack.NewSecretsVerifier(r.Header, h.slackSigningSecret)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text")
		w.Write([]byte(r.Challenge))
	}
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			h.slackClient.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
		}
	}
}
