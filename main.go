package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

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
	ctx := r.Context()

	// Read body.
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Read body: %v.", err)
		return
	}

	// Validating a request.
	if err := validateRequest(h.slackSigningSecret, r.Header, bodyBytes); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("Validate request: %v.", err)
		return
	}

	eventsAPIEvent, err := slackevents.ParseEvent(bodyBytes, slackevents.OptionNoVerifyToken())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Parse event: %v.", err)
		return
	}

	switch eventsAPIEvent.Type {
	case slackevents.URLVerification:
		var r slackevents.ChallengeResponse
		if err := json.Unmarshal(bodyBytes, &r); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Unmarshal challenge response: %v.", err)
			return
		}

		w.Header().Set("Content-type", "text/plain")
		w.Write([]byte(r.Challenge))
	case slackevents.CallbackEvent:
		switch e := eventsAPIEvent.InnerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			if strings.Contains(e.Text, "skip") {
				h.slackClient.PostMessageContext(ctx, e.Channel, slack.MsgOptionText("Skip duty", false))
			} else {
				dh, err := h.dutyHistoryRepo.GetLatestDutyMember(ctx)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					log.Printf("Get latest duty member: %v.", err)
					return
				}

				m, err := h.memberRepo.Get(ctx, dh.MemberID)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					log.Printf("Get member: %v.", err)
					return
				}

				h.slackClient.PostMessageContext(ctx, e.Channel, slack.MsgOptionText(fmt.Sprintf("<@%s> You're today's duty.", m.SlackID), false))
			}
		}
	}
}

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
