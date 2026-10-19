package api

import (
	"net/http"
	"sync"

	"substation-license-server/internal/config"
)

// livePrice lets the owner page change the price without a restart (a restart
// would sign the owner out). Empty means "use the configured price".
type livePrice struct {
	mu     sync.RWMutex
	amount string
}

// price returns the amount and currency every checkout, page and invoice uses.
func (s *Server) price() (string, string) {
	s.livePrice.mu.RLock()
	defer s.livePrice.mu.RUnlock()
	if s.livePrice.amount != "" {
		return s.livePrice.amount, s.Config.LicenseCurrency
	}
	return s.Config.LicensePrice, s.Config.LicenseCurrency
}

// ownerSetPrice saves the new price to config.json (so it survives restarts) and
// uses it at once. It needs a fresh authenticator code: a stolen session must not
// be able to make keys cheap.
func (s *Server) ownerSetPrice(r *http.Request, call *ownerCall) (any, error) {
	var request struct {
		Price string `json:"price"`
	}
	if err := decode(r, &request); err != nil {
		return nil, err
	}
	amount, err := config.NormalizePrice(request.Price)
	if err != nil {
		return nil, badRequest("Write the price as a number, for example 149 or 149.00.")
	}
	old, _ := s.price()
	saved, err := config.LoadFile(s.Config.DataDir)
	if err != nil {
		return nil, err
	}
	saved.LicensePrice = amount
	if err := saved.Save(); err != nil {
		return nil, err
	}
	s.livePrice.mu.Lock()
	s.livePrice.amount = amount
	s.livePrice.mu.Unlock()
	s.ownerAudit("price", call.ip, old+" → "+amount, true)
	return map[string]any{"price": amount, "amount": s.data().Amount()}, nil
}
