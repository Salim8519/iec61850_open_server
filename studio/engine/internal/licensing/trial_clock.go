package licensing

import (
	"strconv"

	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// TrialClock remembers the license server's signed time for trials, and when the anchor
// went missing (licensing/trial_clock.py). Only genuine stamps are ever used; an
// unreadable or foreign copy is ignored, never held against the user. Both values live in
// three places (ClockValue).
type TrialClock struct {
	Stamps  ClockValue // "server_time": the newest signed stamp, as JSON text
	Missing ClockValue // "grace_start": when every anchor copy was noticed gone
	Keys    map[string][]byte
}

// NewTrialClock returns the trial clock kept under home.
func NewTrialClock(home string, registry Registry, keys map[string][]byte) TrialClock {
	return TrialClock{Stamps: NewClockValue("server_time", home, registry),
		Missing: NewClockValue("grace_start", home, registry), Keys: keys}
}

// Confirmed is the time of the newest genuine stamp in any place, or 0.
func (c TrialClock) Confirmed() (int64, error) {
	var newest int64
	for _, copy := range c.Stamps.ReadAll() {
		moment, ok, err := VerifyServerTime(copy, c.Keys)
		if err != nil {
			return 0, err
		}
		if ok {
			newest = max(newest, moment)
		}
	}
	return newest, nil
}

// Remember stores a genuine stamp everywhere when it is not older than the newest one,
// and clears the missing-anchor mark; ok is false when stamp is not genuine.
func (c TrialClock) Remember(stamp any) (bool, error) {
	moment, ok, err := VerifyServerTime(stamp, c.Keys)
	if err != nil || !ok {
		return false, err
	}
	confirmed, err := c.Confirmed()
	if err != nil {
		return false, err
	}
	if moment >= confirmed {
		value := stamp
		if text, isText := stamp.(string); isText {
			if value, err = pytext.LoadsString(text); err != nil {
				return false, uncaught(err.Error())
			}
		}
		data, err := pyjson.Dumps(value)
		if err != nil {
			return false, err
		}
		c.Stamps.Write(string(data))
	}
	if c.SuspiciousSince() != 0 {
		c.Missing.Write("0")
	}
	return true, nil
}

// SuspiciousSince is when this PC noticed every anchor copy gone (0: never, or cleared by
// a stamp): the smallest positive copy.
func (c TrialClock) SuspiciousSince() int64 {
	var earliest int64
	for _, number := range c.Missing.ReadInts() {
		if number.Sign() > 0 && (earliest == 0 || saturate(number) < earliest) {
			earliest = saturate(number)
		}
	}
	return earliest
}

// MarkSuspicious records now as the moment the anchor went missing, once.
func (c TrialClock) MarkSuspicious(now int64) {
	if c.SuspiciousSince() == 0 {
		c.Missing.Write(strconv.FormatInt(now, 10))
	}
}
