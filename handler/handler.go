package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/178inaba/today-duty-bot/repository"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type Handler struct {
	memberRepo         *repository.MemberRepository
	dutyHistoryRepo    *repository.DutyHistoryRepository
	slackClient        *slack.Client
	slackSigningSecret string
}

func NewHandler(
	memberRepo *repository.MemberRepository,
	dutyHistoryRepo *repository.DutyHistoryRepository,
	slackClient *slack.Client,
	slackSigningSecret string,
) *Handler {
	return &Handler{
		memberRepo:         memberRepo,
		dutyHistoryRepo:    dutyHistoryRepo,
		slackClient:        slackClient,
		slackSigningSecret: slackSigningSecret,
	}
}

func (h *Handler) ReceiveEvent(w http.ResponseWriter, r *http.Request) {
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
