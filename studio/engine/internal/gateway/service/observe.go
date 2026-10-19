package service

import (
	"math/big"
	"time"

	"studio-engine/internal/pyjson"
)

// ObserveFor is how long a command waits to see the new position (OBSERVE_SECONDS).
const ObserveFor = 3 * time.Second

// observe records the status the IED reports after an operated command. Python polled the
// signal index every 100 ms; here the check runs at once and then on every hub change of
// that status, so the log says "is now ..." as soon as the value arrives.
func (e *executor) observe(seq int64, iedID, statusRef string, expected any) {
	m := e.manager
	changed := make(chan struct{}, 1)
	cancel := m.runtime.Subscribe(func(change Change) {
		if change.IEDID == "" || (change.IEDID == iedID && (change.Ref == "" || change.Ref == statusRef)) {
			select {
			case changed <- struct{}{}:
			default:
			}
		}
	})
	defer cancel()
	deadline := time.NewTimer(ObserveFor)
	defer deadline.Stop()
	for {
		var current any
		if signal, found := m.runtime.Signal(iedID, statusRef); found {
			current = signal.Value
		}
		if sameTypeAndValue(current, expected) {
			m.Commands.Update(seq, pyjson.NewObject().Set("observed", statusRef+" is now "+PositionText(current)))
			return
		}
		select {
		case <-changed:
		case <-deadline.C:
			m.Commands.Update(seq, pyjson.NewObject().Set("observed", statusRef+" did not change within 3 s"))
			return
		}
	}
}

// sameTypeAndValue is `type(current) is type(expected) and current == expected` for a
// bool or an int expected value.
func sameTypeAndValue(current, expected any) bool {
	switch want := expected.(type) {
	case bool:
		got, ok := current.(bool)
		return ok && got == want
	case int64:
		switch got := current.(type) {
		case int64:
			return got == want
		case int:
			return int64(got) == want
		case *big.Int:
			return got.IsInt64() && got.Int64() == want
		}
	}
	return false
}
