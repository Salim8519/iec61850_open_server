package config

import (
	"errors"
	"strconv"
	"strings"
)

// NormalizePrice turns "149", "149.5" or "0.1" into PayPal's two-decimal form
// ("149.00", "149.50", "0.10"). Captured amounts are compared as text, so a price
// saved without decimals would never match a real payment.
func NormalizePrice(text string) (string, error) {
	text = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "$"))
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || value < 0.01 || value > 100000 {
		return "", errors.New("price must be a number between 0.01 and 100000, for example 149 or 149.00")
	}
	return strconv.FormatFloat(value, 'f', 2, 64), nil
}

// DataBudgetBytes is the most disk the data folder may use before new free trials pause.
func (c Config) DataBudgetBytes() int64 {
	if c.MaxDataMB <= 0 {
		return 45 << 20
	}
	return int64(c.MaxDataMB) << 20
}
