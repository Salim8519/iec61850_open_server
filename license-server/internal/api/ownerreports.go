package api

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"substation-license-server/internal/sales"
)

// money adds amounts per currency in cents, so totals never pick up float errors.
type money map[string]int64

func (m money) add(currency, amount string) {
	value, err := strconv.ParseFloat(amount, 64)
	if err == nil {
		m[currency] += int64(value*100 + 0.5)
	}
}

func (m money) text() map[string]string {
	out := map[string]string{}
	for currency, cents := range m {
		out[currency] = strconv.FormatFloat(float64(cents)/100, 'f', 2, 64)
	}
	return out
}

type salesWindow struct {
	Sales   int               `json:"sales"`
	Refunds int               `json:"refunds"`
	Revenue map[string]string `json:"revenue"`
}

func summarize(records []sales.Record, since int64) salesWindow {
	paid, refunded := money{}, money{}
	window := salesWindow{}
	for _, record := range records {
		if record.Time < since || record.Mode == "sandbox" {
			continue
		}
		switch record.Event {
		case "paid":
			window.Sales++
			paid.add(record.Currency, record.Amount)
		case "refunded", "reversed":
			window.Refunds++
			refunded.add(record.Currency, record.Amount)
		}
	}
	for currency, cents := range refunded {
		paid[currency] -= cents
	}
	window.Revenue = paid.text()
	return window
}

func (s *Server) ownerOverview(r *http.Request, _ *ownerCall) (any, error) {
	licenses, err := s.Store.ListLicenses()
	if err != nil {
		return nil, err
	}
	counts := map[string]int{"total": len(licenses)}
	for _, license := range licenses {
		switch {
		case license.Revoked:
			counts["revoked"]++
		case license.HWID != "":
			counts["active"]++
		default:
			counts["waiting"]++
		}
		if license.OrderID == "" {
			counts["hand_made"]++
		} else {
			counts["sold"]++
		}
	}
	records, err := sales.Read(s.Config.DataDir)
	if err != nil {
		return nil, err
	}
	now := s.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()
	subscribers, _ := s.Store.ListSubscribers()
	active := 0
	for _, subscriber := range subscribers {
		if subscriber.Active() {
			active++
		}
	}
	recent, _ := s.owner.audit.Recent(8)
	return map[string]any{
		"licenses": counts, "trials": s.Store.TrialCount(), "subscribers": active,
		"sales": map[string]salesWindow{"today": summarize(records, today), "month": summarize(records, month),
			"all": summarize(records, 0)},
		"price": s.data().Amount(), "checkout": s.Config.PayPalReady(), "paypal_mode": s.Config.PayPalMode,
		"mail": s.Mailer != nil, "data_mb": float64(FolderBytes(s.Config.DataDir)) / (1 << 20),
		"data_budget_mb": s.Config.DataBudgetBytes() >> 20, "recent": recent, "server_time": s.Now().Unix(),
	}, nil
}

func (s *Server) ownerPurchases(r *http.Request, _ *ownerCall) (any, error) {
	records, err := sales.Read(s.Config.DataDir)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(records, func(i, j int) bool { return records[i].Time > records[j].Time })
	if len(records) > 500 {
		records = records[:500]
	}
	if records == nil {
		records = []sales.Record{}
	}
	return map[string]any{"records": records}, nil
}

func (s *Server) ownerActivity(r *http.Request, _ *ownerCall) (any, error) {
	entries, err := s.owner.audit.Recent(300)
	return map[string]any{"entries": entries, "sessions": s.owner.sessions.Count()}, err
}
