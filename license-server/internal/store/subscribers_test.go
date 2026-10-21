package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSubscribersNeedConsentAndCanLeave(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Unix(1_800_000_000, 0)
	if _, err := db.Subscribe("not an email", "owner", now); !errors.Is(err, ErrBadEmail) {
		t.Fatal("invalid address must be refused")
	}
	first, err := db.Subscribe(" Buyer@Example.com ", "checkout", now)
	if err != nil || first.Email != "buyer@example.com" || len(first.Token) != 32 {
		t.Fatalf("subscribe failed %+v %v", first, err)
	}
	if found, _ := db.Unsubscribe(first.Token, now); !found {
		t.Fatal("token must unsubscribe")
	}
	if _, err := db.Subscribe("buyer@example.com", "owner", now); !errors.Is(err, ErrUnsubscribed) {
		t.Fatal("the owner must not re-add someone who left")
	}
	again, err := db.Subscribe("buyer@example.com", "checkout", now)
	if err != nil || !again.Active() || again.Token != first.Token {
		t.Fatal("a new checkout consent may re-add with the same token")
	}
	if found, _ := db.Unsubscribe("short", now); found {
		t.Fatal("bad token must not match")
	}
	db.RemoveSubscriber("BUYER@example.com")
	if list, _ := db.ListSubscribers(); len(list) != 0 {
		t.Fatal("remove must delete the record")
	}
}
