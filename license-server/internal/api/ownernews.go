package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"substation-license-server/internal/store"
)

// NewsMailer sends one news email with an unsubscribe link (faked in tests).
type NewsMailer interface {
	SendNews(ctx context.Context, to, subject, text, unsubscribeURL string) error
}

// newsJob is the one send that may run at a time; the page polls its progress.
type newsJob struct {
	mu       sync.Mutex
	Running  bool   `json:"running"`
	Subject  string `json:"subject"`
	Total    int    `json:"total"`
	Sent     int    `json:"sent"`
	Failed   int    `json:"failed"`
	Started  int64  `json:"started"`
	Finished int64  `json:"finished"`
}

func (j *newsJob) snapshot() map[string]any {
	j.mu.Lock()
	defer j.mu.Unlock()
	return map[string]any{"running": j.Running, "subject": j.Subject, "total": j.Total, "sent": j.Sent,
		"failed": j.Failed, "started": j.Started, "finished": j.Finished}
}

type newsRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Email   string `json:"email"`
	Consent bool   `json:"consent"`
}

func readNews(r *http.Request) (newsRequest, error) {
	var request newsRequest
	if err := decode(r, &request); err != nil {
		return request, err
	}
	request.Subject, request.Body = strings.TrimSpace(request.Subject), strings.TrimSpace(request.Body)
	if len([]rune(request.Subject)) < 3 || len([]rune(request.Subject)) > 150 || strings.ContainsAny(request.Subject, "\r\n") {
		return request, badRequest("Write a one-line subject of 3 to 150 characters.")
	}
	if len([]rune(request.Body)) < 10 || len([]rune(request.Body)) > 20000 {
		return request, badRequest("Write a message of 10 to 20,000 characters.")
	}
	return request, nil
}

func (s *Server) unsubscribeURL(token string) string {
	return strings.TrimRight(s.Config.PublicURL, "/") + "/unsubscribe?t=" + url.QueryEscape(token)
}

func newsText(body, unsubscribe string) string {
	return body + "\n\n--\nYou get this email because you asked for ScadaBench news.\n" +
		"Unsubscribe with one click: " + unsubscribe + "\n"
}

func (s *Server) ownerSubscribers(r *http.Request, _ *ownerCall) (any, error) {
	subscribers, err := s.Store.ListSubscribers()
	active := 0
	for _, subscriber := range subscribers {
		if subscriber.Active() {
			active++
		}
	}
	for i := range subscribers {
		subscribers[i].Token = ""
	}
	return map[string]any{"subscribers": subscribers, "active": active, "ready": s.News != nil,
		"from": s.newsFrom(), "job": s.owner.news.snapshot(), "test_to": s.Config.NotifyEmail}, err
}

func (s *Server) newsFrom() string {
	if s.Config.MarketingFrom != "" {
		return s.Config.MarketingFrom
	}
	return s.Config.MailgunFrom
}

func (s *Server) ownerAddSubscriber(r *http.Request, call *ownerCall) (any, error) {
	var request newsRequest
	if err := decode(r, &request); err != nil {
		return nil, err
	}
	if !request.Consent {
		return nil, badRequest("Only add people who agreed to get news from you. Tick the box to confirm they did.")
	}
	subscriber, err := s.Store.Subscribe(request.Email, "owner", s.Now())
	if err == store.ErrUnsubscribed || err == store.ErrBadEmail {
		return nil, badRequest(err.Error())
	}
	s.ownerAudit("subscriber-add", call.ip, subscriber.Email, err == nil)
	subscriber.Token = ""
	return subscriber, err
}

func (s *Server) ownerRemoveSubscriber(r *http.Request, call *ownerCall) (any, error) {
	var request newsRequest
	if err := decode(r, &request); err != nil {
		return nil, err
	}
	err := s.Store.RemoveSubscriber(request.Email)
	s.ownerAudit("subscriber-remove", call.ip, strings.ToLower(strings.TrimSpace(request.Email)), err == nil)
	return map[string]any{"removed": request.Email}, err
}

func (s *Server) ownerNewsTest(r *http.Request, call *ownerCall) (any, error) {
	request, err := readNews(r)
	if err != nil {
		return nil, err
	}
	if s.News == nil || s.Config.NotifyEmail == "" {
		return nil, apiError{http.StatusServiceUnavailable, "mail_off", "Email or the owner address is not set up on the server."}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	err = s.News.SendNews(ctx, s.Config.NotifyEmail, "[TEST] "+request.Subject,
		newsText(request.Body, s.unsubscribeURL("test")), s.unsubscribeURL("test"))
	s.ownerAudit("news-test", call.ip, request.Subject, err == nil)
	if err != nil {
		s.Log.Event("mail_error", map[string]any{"step": "news_test", "error": err.Error()})
		return nil, apiError{http.StatusBadGateway, "mail_failed", "The test email could not be sent. See Issues."}
	}
	return map[string]any{"sent_to": s.Config.NotifyEmail}, nil
}

func (s *Server) ownerNewsSend(r *http.Request, call *ownerCall) (any, error) {
	request, err := readNews(r)
	if err != nil {
		return nil, err
	}
	if s.News == nil {
		return nil, apiError{http.StatusServiceUnavailable, "mail_off", "Email is not set up on the server."}
	}
	subscribers, err := s.Store.ListSubscribers()
	if err != nil {
		return nil, err
	}
	targets := []store.Subscriber{}
	for _, subscriber := range subscribers {
		if subscriber.Active() {
			targets = append(targets, subscriber)
		}
	}
	job := s.owner.news
	job.mu.Lock()
	if job.Running {
		job.mu.Unlock()
		return nil, apiError{http.StatusConflict, "busy", "A send is already running. Wait for it to finish."}
	}
	job.Running, job.Subject, job.Total, job.Sent, job.Failed = true, request.Subject, len(targets), 0, 0
	job.Started, job.Finished = s.Now().Unix(), 0
	job.mu.Unlock()
	s.ownerAudit("news-send", call.ip, fmt.Sprintf("%q to %d people", request.Subject, len(targets)), true)
	go s.runNews(request, targets)
	return job.snapshot(), nil
}

// runNews sends one email every 300 ms, well under Mailgun's limits.
func (s *Server) runNews(request newsRequest, targets []store.Subscriber) {
	job := s.owner.news
	for _, subscriber := range targets {
		link := s.unsubscribeURL(subscriber.Token)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err := s.News.SendNews(ctx, subscriber.Email, request.Subject, newsText(request.Body, link), link)
		cancel()
		job.mu.Lock()
		if err != nil {
			job.Failed++
			s.Log.Event("mail_error", map[string]any{"step": "news", "error": err.Error()})
		} else {
			job.Sent++
		}
		job.mu.Unlock()
		time.Sleep(300 * time.Millisecond)
	}
	job.mu.Lock()
	job.Running, job.Finished = false, s.Now().Unix()
	job.mu.Unlock()
}

func (s *Server) ownerNewsStatus(r *http.Request, _ *ownerCall) (any, error) {
	return s.owner.news.snapshot(), nil
}
