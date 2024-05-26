package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/178inaba/today-duty-bot/repository"
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

	memberRepo := repository.NewMemberRepository(db)
	dutyHistoryRepo := repository.NewDutyHistoryRepository(db)

	h := newHandler(
		memberRepo,
		dutyHistoryRepo,
		slack.New(os.Getenv("SLACK_TOKEN")),
		os.Getenv("SLACK_SIGNING_SECRET"),
	)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/events", h.receiveEvent)

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

type handler struct {
	memberRepo         *repository.MemberRepository
	dutyHistoryRepo    *repository.DutyHistoryRepository
	slackClient        *slack.Client
	slackSigningSecret string
}

func newHandler(
	memberRepo *repository.MemberRepository,
	dutyHistoryRepo *repository.DutyHistoryRepository,
	slackClient *slack.Client,
	slackSigningSecret string,
) *handler {
	return &handler{
		memberRepo:         memberRepo,
		dutyHistoryRepo:    dutyHistoryRepo,
		slackClient:        slackClient,
		slackSigningSecret: slackSigningSecret,
	}
}

func (h *handler) receiveEvent(w http.ResponseWriter, r *http.Request) {
	// Read body.
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Read body: %v.", err)
		return
	}

	if err := validateRequest(h.slackSigningSecret, r.Header, bodyBytes); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Validate request: %v.", err)
		return
	}

	log.Printf("Receive event: %s.", string(bodyBytes))

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(bodyBytes), slackevents.OptionNoVerifyToken())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal(bodyBytes, &r)
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

// Validating a request.
func validateRequest(signingSecret string, header http.Header, body []byte) error {
	sv, err := slack.NewSecretsVerifier(header, signingSecret)
	if err != nil {
		return fmt.Errorf("new secret verifier: %w", err)
	}
	if _, err := sv.Write(body); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	if err := sv.Ensure(); err != nil {
		return fmt.Errorf("ensure secret: %w", err)
	}

	return nil
}
