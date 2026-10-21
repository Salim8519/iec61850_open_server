package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"substation-license-server/internal/adminauth"
)

// ownerRoutes lists every owner API call and the check level it needs.
func (s *Server) ownerRoutes(mux *http.ServeMux, api string) {
	routes := []struct {
		pattern string
		level   int
		handler ownerHandler
	}{
		{"GET session", public, s.ownerSessionInfo},
		{"POST login", public, s.ownerLogin},
		{"POST logout", signedIn, s.ownerLogout},
		{"POST logout-all", signedIn, s.ownerLogoutAll},
		{"GET overview", signedIn, s.ownerOverview},
		{"POST price", confirmed, s.ownerSetPrice},
		{"GET versions", signedIn, s.ownerVersions},
		{"GET licenses", signedIn, s.ownerLicenses},
		{"POST licenses/issue", signedIn, s.ownerIssue},
		{"POST licenses/note", signedIn, s.ownerNote},
		{"POST licenses/revoke", signedIn, s.ownerRevoke},
		{"POST licenses/resend", signedIn, s.ownerResend},
		{"POST licenses/delete", confirmed, s.ownerDelete},
		{"GET purchases", signedIn, s.ownerPurchases},
		{"GET activity", signedIn, s.ownerActivity},
		{"GET issues", signedIn, s.ownerIssues},
		{"GET disk", signedIn, s.ownerDisk},
		{"POST disk/clean", confirmed, s.ownerClean},
		{"GET subscribers", signedIn, s.ownerSubscribers},
		{"POST subscribers/add", signedIn, s.ownerAddSubscriber},
		{"POST subscribers/remove", signedIn, s.ownerRemoveSubscriber},
		{"POST news/test", signedIn, s.ownerNewsTest},
		{"POST news/send", confirmed, s.ownerNewsSend},
		{"GET news/status", signedIn, s.ownerNewsStatus},
	}
	for _, route := range routes {
		method, path, _ := strings.Cut(route.pattern, " ")
		mux.HandleFunc(method+" "+api+path, s.ownerAPI(route.level, route.handler))
	}
}

func (s *Server) ownerSessionInfo(r *http.Request, call *ownerCall) (any, error) {
	_, ready, err := s.owner.keeper.Load()
	if err != nil {
		return nil, err
	}
	signed := s.ownerSession(r, call)
	result := map[string]any{"signed_in": signed, "setup_done": ready}
	if signed {
		result["csrf"] = call.session.CSRF
		result["expires_at"] = call.session.Created.Add(adminauth.SessionMax).Unix()
	}
	return result, nil
}

// ownerLogin checks password and second step together, so a wrong try never says
// which one was wrong. One check runs at a time to bound CPU on the small server.
func (s *Server) ownerLogin(r *http.Request, call *ownerCall) (any, error) {
	if locked, until := s.owner.lockout.Locked(s.Now()); locked {
		return nil, apiError{http.StatusLocked, "locked", "Sign-in is locked after too many wrong tries. Try again after " +
			until.UTC().Format("15:04") + " UTC, or reset it from your PC with reset-admin.cmd."}
	}
	if !s.owner.limiter.Allow(call.ip, 1, s.Now()) {
		return nil, apiError{http.StatusTooManyRequests, "rate_limited", "Too many tries. Wait a few minutes."}
	}
	var request struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := decode(r, &request); err != nil {
		return nil, err
	}
	s.owner.loginMu <- struct{}{}
	defer func() { <-s.owner.loginMu }()
	account, ready, err := s.owner.keeper.Load()
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, apiError{http.StatusServiceUnavailable, "no_account", "The owner login is not set up yet. Run admin-setup.cmd on your PC."}
	}
	factor := adminauth.FactorNone
	if adminauth.CheckPassword(account.PasswordHash, request.Password) {
		account, _, err = s.owner.keeper.Update(func(a *adminauth.Account) bool {
			factor = a.CheckSecondFactor(request.Code, s.Now())
			return factor != adminauth.FactorNone
		})
		if err != nil {
			return nil, err
		}
	}
	if factor == adminauth.FactorNone {
		s.owner.limiter.Fail(call.ip, s.Now())
		s.ownerAudit("login", call.ip, "wrong password or code", false)
		if s.owner.lockout.Fail(s.Now()) {
			s.ownerAudit("lockout", call.ip, "sign-in locked for 1 hour", false)
			s.ownerAlert("Owner page locked after wrong sign-in tries",
				"Sign-in to the ScadaBench owner page was locked for one hour after 10 wrong tries.\n"+
					"Last try from: "+call.ip+"\nIf this was not you, nothing else is needed: the lock protects the page.")
		}
		return nil, apiError{http.StatusUnauthorized, "wrong_login", "Wrong password or code. Check the time on your phone if codes keep failing."}
	}
	s.owner.lockout.Succeed()
	token, session := s.owner.sessions.Create(account.Epoch, call.ip, s.Now())
	call.session, call.token = session, token
	detail := "authenticator code"
	if factor == adminauth.FactorRecovery {
		detail = fmt.Sprintf("recovery code (%d left)", len(account.RecoveryHashes))
	}
	s.ownerAudit("login", call.ip, detail, true)
	s.ownerAlert("New sign-in to the owner page", "Someone signed in to the ScadaBench owner page.\n\n"+
		"From: "+call.ip+"\nWith: "+detail+"\nTime (UTC): "+s.Now().UTC().Format("2006-01-02 15:04")+
		"\n\nNot you? Run reset-admin.cmd on your PC now: it changes the password and signs everyone out.")
	return ownerCookieResult{token: token, csrf: session.CSRF, path: s.Config.AdminPath}, nil
}

// ownerCookieResult is written by writeOwnerLogin so the cookie and body go out together.
type ownerCookieResult struct{ token, csrf, path string }

func (s *Server) ownerLogout(r *http.Request, call *ownerCall) (any, error) {
	s.owner.sessions.Delete(call.token)
	s.ownerAudit("logout", call.ip, "", true)
	return ownerCookieResult{path: s.Config.AdminPath}, nil
}

func (s *Server) ownerLogoutAll(r *http.Request, call *ownerCall) (any, error) {
	count := s.owner.sessions.DeleteAll()
	s.ownerAudit("logout-all", call.ip, fmt.Sprintf("%d sessions ended", count), true)
	return ownerCookieResult{path: s.Config.AdminPath}, nil
}

// setOwnerCookie sets (token != "") or clears the session cookie.
func setOwnerCookie(w http.ResponseWriter, result ownerCookieResult) {
	cookie := &http.Cookie{Name: ownerCookie, Value: result.token, Path: result.path, Secure: true,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: int(adminauth.SessionMax / time.Second)}
	if result.token == "" {
		cookie.MaxAge = -1
	}
	http.SetCookie(w, cookie)
}

// ownerAlert emails the owner in the background; failures only reach the log.
func (s *Server) ownerAlert(subject, text string) {
	if s.Mailer == nil || s.Config.NotifyEmail == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.Mailer.Send(ctx, s.Config.NotifyEmail, "[ScadaBench] "+subject, text); err != nil {
			s.Log.Event("mail_error", map[string]any{"step": "owner_alert", "error": err.Error()})
		}
	}()
}
