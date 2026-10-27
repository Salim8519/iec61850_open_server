package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"substation-license-server/internal/paypal"
)

type fakeNews struct {
	mu   sync.Mutex
	sent []string
}

func (f *fakeNews) SendNews(_ context.Context, to, subject, text, unsubscribe string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, to+"|"+subject+"|"+text+"|"+unsubscribe)
	return nil
}

func TestOwnerMakesFriendKeyAndConfirmsDelete(t *testing.T) {
	h := newOwnerHarness(t)
	mailer := &fakeMail{}
	h.s.Mailer = mailer
	h.login()
	status, result, _ := h.do("POST", "/owner-test/api/licenses/issue",
		map[string]any{"email": "ahmed@example.com", "note": "Ahmed, friend", "send_email": true}, nil)
	license, _ := result["license"].(map[string]any)
	key, _ := license["key"].(string)
	if status != 200 || !strings.HasPrefix(key, "SUBS-") || license["note"] != "Ahmed, friend" || result["emailed"] != true {
		t.Fatalf("issue failed %d %v", status, result)
	}
	if len(mailer.sent) != 1 || !strings.Contains(mailer.sent[0], key) {
		t.Fatal("the key must be emailed to the friend")
	}
	if _, found, _ := h.do("GET", "/owner-test/api/licenses?q=ahmed", nil, nil); found["total"] != float64(1) {
		t.Fatalf("search must find the key %v", found)
	}
	if status, result, _ := h.do("POST", "/owner-test/api/licenses/delete", map[string]string{"key": key}, nil); status != 403 ||
		result["code"] != "confirm" {
		t.Fatal("delete must ask for a fresh authenticator code")
	}
	if status, _, _ := h.do("POST", "/owner-test/api/licenses/delete", map[string]string{"key": key},
		map[string]string{confirmHeader: h.code()}); status != 200 {
		t.Fatal("delete with a fresh code must work")
	}
	if _, err := h.s.Store.GetLicense(key); err == nil {
		t.Fatal("key must be gone")
	}
}

func TestOwnerDailyKeyCap(t *testing.T) {
	h := newOwnerHarness(t)
	h.login()
	for i := 0; i < ownerDailyKeys; i++ {
		if status, _, _ := h.do("POST", "/owner-test/api/licenses/issue", map[string]any{}, nil); status != 200 {
			t.Fatalf("key %d refused", i)
		}
	}
	if status, _, _ := h.do("POST", "/owner-test/api/licenses/issue", map[string]any{}, nil); status != http.StatusTooManyRequests {
		t.Fatal("the 26th key in a day must be refused")
	}
}

func TestCheckoutOptInSubscribesAndOneClickUnsubscribe(t *testing.T) {
	h := newOwnerHarness(t)
	invoice := h.s.newInvoice("149.00", "USD", true)
	if !wantsNews(invoice) || !h.s.invoiceQuoted(invoice, "149.00", "USD") || wantsNews(h.s.newInvoice("1.00", "USD", false)) {
		t.Fatal("opt-in must ride in the signed invoice id")
	}
	_, err := h.s.Fulfil(context.Background(), paypal.Capture{Status: "COMPLETED", OrderID: "O-1", CaptureID: "C-1",
		Email: "buyer@example.com", Amount: h.s.Config.LicensePrice, Currency: h.s.Config.LicenseCurrency, InvoiceID: invoice})
	if err != nil {
		t.Fatal(err)
	}
	subscribers, _ := h.s.Store.ListSubscribers()
	if len(subscribers) != 1 || subscribers[0].Source != "checkout" {
		t.Fatalf("buyer who ticked the box must be subscribed %+v", subscribers)
	}
	page := httptest.NewRecorder()
	h.handler.ServeHTTP(page, httptest.NewRequest("GET", "/unsubscribe?t="+subscribers[0].Token, nil))
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Unsubscribe</button>") {
		t.Fatal("GET must only show the button")
	}
	if list, _ := h.s.Store.ListSubscribers(); !list[0].Active() {
		t.Fatal("GET must not unsubscribe (mail scanners open links)")
	}
	form := url.Values{"List-Unsubscribe": {"One-Click"}}.Encode()
	post := httptest.NewRequest("POST", "/unsubscribe?t="+subscribers[0].Token, strings.NewReader(form))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	done := httptest.NewRecorder()
	h.handler.ServeHTTP(done, post)
	if list, _ := h.s.Store.ListSubscribers(); done.Code != 200 || list[0].Active() {
		t.Fatal("one-click POST must unsubscribe")
	}
}

func TestOwnerNewsGoesOnlyToSubscribersWithUnsubscribeLink(t *testing.T) {
	h := newOwnerHarness(t)
	news := &fakeNews{}
	h.s.News = news
	h.s.Config.PublicURL = "https://license.example.com"
	h.login()
	if status, _, _ := h.do("POST", "/owner-test/api/subscribers/add", map[string]any{"email": "a@example.com"}, nil); status != 400 {
		t.Fatal("adding a person needs their consent tick")
	}
	h.do("POST", "/owner-test/api/subscribers/add", map[string]any{"email": "a@example.com", "consent": true}, nil)
	h.do("POST", "/owner-test/api/subscribers/add", map[string]any{"email": "b@example.com", "consent": true}, nil)
	h.s.Store.Unsubscribe(mustToken(t, h, "b@example.com"), h.clock)
	body := map[string]string{"subject": "Version 0.2", "body": "Hello, version 0.2 is out."}
	if status, _, _ := h.do("POST", "/owner-test/api/news/send", body, nil); status != 403 {
		t.Fatal("sending needs a fresh code")
	}
	if status, result, _ := h.do("POST", "/owner-test/api/news/send", body, map[string]string{confirmHeader: h.code()}); status != 200 ||
		result["total"] != float64(1) {
		t.Fatalf("send failed %d %v", status, result)
	}
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if h.s.owner.news.snapshot()["running"] == false {
			break
		}
	}
	news.mu.Lock()
	defer news.mu.Unlock()
	if len(news.sent) != 1 || !strings.HasPrefix(news.sent[0], "a@example.com|Version 0.2|") ||
		!strings.Contains(news.sent[0], "https://license.example.com/unsubscribe?t=") {
		t.Fatalf("news must reach only the active subscriber with a link: %v", news.sent)
	}
}

func mustToken(t *testing.T, h *ownerHarness, email string) string {
	list, _ := h.s.Store.ListSubscribers()
	for _, subscriber := range list {
		if subscriber.Email == email {
			return subscriber.Token
		}
	}
	t.Fatal("subscriber missing")
	return ""
}

func TestOwnerCleanRemovesOnlyOldLogs(t *testing.T) {
	h := newOwnerHarness(t)
	h.login()
	old := filepath.Join(h.s.Config.LogDir, "server.log.1")
	os.WriteFile(old, []byte(strings.Repeat("x", 2048)), 0o600)
	status, preview, _ := h.do("POST", "/owner-test/api/disk/clean", map[string]bool{"preview": true}, map[string]string{confirmHeader: h.code()})
	if status != 200 || preview["bytes"] != float64(2048) {
		t.Fatalf("preview wrong %d %v", status, preview)
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatal("preview must not delete")
	}
	h.do("POST", "/owner-test/api/disk/clean", map[string]bool{"preview": false}, map[string]string{confirmHeader: h.code()})
	if _, err := os.Stat(old); err == nil {
		t.Fatal("old log must be removed")
	}
	if _, err := os.Stat(filepath.Join(h.s.Config.DataDir, "licenses.db")); err != nil {
		t.Fatal("the database must never be removed")
	}
	if _, disk, _ := h.do("GET", "/owner-test/api/disk", nil, nil); disk["files"] == nil {
		t.Fatal("disk view must list files")
	}
}

func TestOwnerPriceChangeNeedsCodeAndAppliesAtOnce(t *testing.T) {
	h := newOwnerHarness(t)
	h.login()
	if status, _, _ := h.do("POST", "/owner-test/api/price", map[string]string{"price": "0.01"}, nil); status != 403 {
		t.Fatal("a price change needs a fresh code")
	}
	if status, _, _ := h.do("POST", "/owner-test/api/price", map[string]string{"price": "abc"},
		map[string]string{confirmHeader: h.code()}); status != 400 {
		t.Fatal("a wrong price must be refused")
	}
	status, result, _ := h.do("POST", "/owner-test/api/price", map[string]string{"price": "99"}, map[string]string{confirmHeader: h.code()})
	if status != 200 || result["price"] != "99.00" {
		t.Fatalf("price change failed %d %v", status, result)
	}
	if amount, _ := h.s.price(); amount != "99.00" || !h.s.paidAmountAccepted("", "99.00", "USD") {
		t.Fatal("checkout must use the new price at once")
	}
	raw, _ := os.ReadFile(filepath.Join(h.s.Config.DataDir, "config.json"))
	if !strings.Contains(string(raw), `"license_price": "99.00"`) {
		t.Fatal("the new price must survive a restart")
	}
	h.s.Config.LemonCheckoutURL, h.s.Config.LemonVariantID = "https://s.lemonsqueezy.com/buy/x", "1"
	h.s.Config.LemonSigningSecret = "a-signing-secret-of-enough-length"
	status, result, _ = h.do("POST", "/owner-test/api/price", map[string]string{"price": "0.10"}, map[string]string{confirmHeader: h.code()})
	if status != 400 || !strings.Contains(fmt.Sprint(result), "0.50") {
		t.Fatalf("with card checkout on, a price under $0.50 must be refused: %d %v", status, result)
	}
}
