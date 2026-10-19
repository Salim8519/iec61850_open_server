package store

import (
	"errors"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketTrialParts = []byte("trial_parts")

	// ErrEmailRequired means the request looks like a repeat (2 of 3 parts, or a busy
	// network): the trial needs a confirmed email address, one trial per address.
	ErrEmailRequired = errors.New("please confirm your email address to start the free trial")
	// ErrEmailUsed means this (normalised) address already had a free trial.
	ErrEmailUsed = errors.New("this email address already had a free trial")
)

// maxMachinesPerPart bounds one part's index entry; a cloned lab image shared by
// many machines must not grow a record without limit.
const maxMachinesPerPart = 50

// TrialRefused explains a refusal or an extra step, for the server log.
type TrialRefused struct {
	Cause   error
	Reason  string // same_machine, hardware_match, needs_email or email_used
	Matched string // why: matched part kinds ("guid+volume") and/or "busy_ip"
}

func (e *TrialRefused) Error() string { return e.Cause.Error() }
func (e *TrialRefused) Unwrap() error { return e.Cause }

// TrialRequest is everything the server knows when a machine asks for a trial.
type TrialRequest struct {
	HWID      string
	Parts     TrialParts
	IP        string
	EmailHash string // set only after the address was confirmed (code or mail outage)
	TrialDays int
	// More than IPEmailAfter new trials from one IP within IPWindow is a suspicion
	// signal (an email step), never a refusal. IPEmailAfter <= 0 turns it off.
	IPEmailAfter int
	IPWindow     time.Duration
}

// StartTrial grants the one trial per machine (old clients: machine ID only).
func (s *Store) StartTrial(hwid, ip string, now time.Time, days int) (Trial, error) {
	trial, _, err := s.StartTrialFor(TrialRequest{HWID: hwid, IP: ip, TrialDays: days}, now)
	return trial, err
}

// StartTrialFor applies the trial rules in one transaction ("smart, not soft"):
//   - same machine ID: the same trial back while it runs, refused once it ended;
//   - every usable part (at least two) matches an ended trial: refused, clear proof;
//   - same hardware with a running trial: that trial continues under the new ID;
//   - 2 of 3 parts match an ended trial, or a busy network: confirm an email first.
//
// Outcomes: "existing", "continued" or "new".
func (s *Store) StartTrialFor(request TrialRequest, now time.Time) (Trial, string, error) {
	var trial Trial
	outcome := "new"
	parts := request.Parts.Clean()
	err := s.db.Update(func(tx *bolt.Tx) error {
		found, err := get(tx, bucketTrials, request.HWID, &trial)
		if err != nil {
			return err
		}
		if found {
			if now.Unix() < trial.ExpiresAt {
				outcome = "existing"
				return nil
			}
			return &TrialRefused{Cause: ErrTrialUsed, Reason: "same_machine"}
		}
		match, err := matchingTrial(tx, request.HWID, parts, now)
		if err != nil {
			return err
		}
		if match.running != nil {
			// The machine ID changed (for example a firmware update) but the hardware is
			// the same: keep the running trial, with its original end date.
			outcome = "continued"
			trial = Trial{HWID: request.HWID, IssuedAt: match.running.IssuedAt, ExpiresAt: match.running.ExpiresAt,
				IP: request.IP, Parts: parts}
			return saveTrial(tx, trial)
		}
		if match.full != "" {
			return &TrialRefused{Cause: ErrTrialUsed, Reason: "hardware_match", Matched: match.full}
		}
		signals := []string{}
		if match.partial != "" {
			signals = append(signals, match.partial)
		}
		if busy, err := busyNetwork(tx, request.IP, request.IPEmailAfter, request.IPWindow, now); err != nil {
			return err
		} else if busy {
			signals = append(signals, "busy_ip")
		}
		if len(signals) > 0 && request.EmailHash == "" {
			return &TrialRefused{Cause: ErrEmailRequired, Reason: "needs_email", Matched: strings.Join(signals, ",")}
		}
		if request.EmailHash != "" {
			if used, err := emailUsed(tx, request.EmailHash); err != nil {
				return err
			} else if used {
				return &TrialRefused{Cause: ErrEmailUsed, Reason: "email_used", Matched: strings.Join(signals, ",")}
			}
			if err := recordEmail(tx, request.EmailHash, request.HWID, now); err != nil {
				return err
			}
		}
		if err := recordNetworkTrial(tx, request.IP, request.IPWindow, now); err != nil {
			return err
		}
		expires := now.Add(time.Duration(request.TrialDays) * 24 * time.Hour).Unix()
		trial = Trial{HWID: request.HWID, IssuedAt: now.Unix(), ExpiresAt: expires, IP: request.IP, Parts: parts,
			Email: request.EmailHash != ""}
		return saveTrial(tx, trial)
	})
	var refused *TrialRefused
	if errors.As(err, &refused) && refused.Reason != "same_machine" {
		s.db.Update(func(tx *bolt.Tx) error { return put(tx, bucketTrialRefusals, request.HWID, parts) })
	}
	return trial, outcome, err
}

func saveTrial(tx *bolt.Tx, trial Trial) error {
	if err := indexParts(tx, trial.HWID, trial.Parts); err != nil {
		return err
	}
	return put(tx, bucketTrials, trial.HWID, trial)
}

type partMatch struct {
	running *Trial // a still-running trial sharing at least two parts
	full    string // kinds, when every usable part (>= 2) matches one ended trial
	partial string // kinds, when at least two (but not all) parts match an ended trial
}

// matchingTrial compares the request's usable parts with earlier trials of other
// machine IDs. One shared part alone is never a signal (cloned VMs, reinstalls).
func matchingTrial(tx *bolt.Tx, hwid string, parts TrialParts, now time.Time) (partMatch, error) {
	var result partMatch
	hits := map[string][]string{}
	for _, part := range parts.entries() {
		var machines []string
		if _, err := get(tx, bucketTrialParts, part.key, &machines); err != nil {
			return result, err
		}
		for _, machine := range machines {
			if machine != hwid {
				hits[machine] = append(hits[machine], part.kind)
			}
		}
	}
	usable := parts.Count()
	for machine, kinds := range hits {
		if len(kinds) < 2 {
			continue
		}
		var earlier Trial
		found, err := get(tx, bucketTrials, machine, &earlier)
		if err != nil {
			return result, err
		}
		if !found {
			continue
		}
		matched := strings.Join(kinds, "+")
		switch {
		case now.Unix() < earlier.ExpiresAt:
			if result.running == nil || earlier.ExpiresAt > result.running.ExpiresAt {
				held := earlier
				result.running = &held
			}
		case len(kinds) == usable:
			result.full = matched
		default:
			result.partial = matched
		}
	}
	return result, nil
}

func indexParts(tx *bolt.Tx, hwid string, parts TrialParts) error {
	for _, part := range parts.entries() {
		var machines []string
		if _, err := get(tx, bucketTrialParts, part.key, &machines); err != nil {
			return err
		}
		if contains(machines, hwid) || len(machines) >= maxMachinesPerPart {
			continue
		}
		if err := put(tx, bucketTrialParts, part.key, append(machines, hwid)); err != nil {
			return err
		}
	}
	return nil
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
