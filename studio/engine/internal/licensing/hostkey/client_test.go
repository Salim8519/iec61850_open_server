package hostkey

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeHost answers on its own pipe: dial hands the client one end and the host the other.
func fakeHost(answer func(Request) Answer, delay time.Duration) Dialer {
	return func(ctx context.Context, address string) (net.Conn, error) {
		client, host := net.Pipe()
		go func() {
			defer host.Close()
			buffer := make([]byte, MaxRequestBytes)
			count, err := host.Read(buffer)
			if err != nil {
				return
			}
			var got Request
			if json.Unmarshal(buffer[:count], &got) != nil {
				return
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
			data, _ := json.Marshal(answer(got))
			host.Write(append(data, '\n'))
		}()
		return client, nil
	}
}

func routed(dialers map[string]Dialer) Dialer {
	return func(ctx context.Context, address string) (net.Conn, error) {
		if dial, found := dialers[address]; found {
			return dial(ctx, address)
		}
		return nil, errors.New("connection refused")
	}
}

func always(signature string) func(Request) Answer {
	return func(Request) Answer { return Answer{Certificate: json.RawMessage(`{}`), Signature: signature} }
}

func accepting(signature string) func(Answer) error {
	return func(a Answer) error {
		if a.Signature != signature {
			return errors.New("not the host")
		}
		return nil
	}
}

func TestTheFirstHostWhoseAnswerIsAcceptedWinsEvenWhenOthersAnswerBadly(t *testing.T) {
	dial := routed(map[string]Dialer{
		"10.0.0.1:47862": fakeHost(always("liar"), 0),
		"10.0.1.1:47862": fakeHost(always("real"), 20*time.Millisecond),
	})
	found, err := Ask(context.Background(), []string{"10.0.0.1", "10.0.1.1", "10.0.2.1"}, request, 2*time.Second, dial,
		accepting("real"))
	if err != nil || found.Host != "10.0.1.1" || found.Answer.Signature != "real" {
		t.Fatalf("found %+v, err %v", found, err)
	}
}

func TestEveryAskedHostGetsTheSameFreshRequest(t *testing.T) {
	seen := make(chan Request, 4)
	record := func(r Request) Answer { seen <- r; return Answer{Signature: "x"} }
	dial := routed(map[string]Dialer{"10.0.0.1:47862": fakeHost(record, 0), "10.0.1.1:47862": fakeHost(record, 0)})
	Ask(context.Background(), []string{"10.0.0.1", "10.0.1.1"}, request, time.Second, dial, func(Answer) error { return errors.New("no") })
	for index := 0; index < 2; index++ {
		if got := <-seen; got != request {
			t.Fatalf("a host got %+v", got)
		}
	}
}

func TestNobodyAnsweringIsAPlainErrorNotAHang(t *testing.T) {
	started := time.Now()
	_, err := Ask(context.Background(), []string{"10.0.0.1"}, request, 150*time.Millisecond, routed(nil), accepting("x"))
	if err == nil || !strings.Contains(err.Error(), "no host confirmed") {
		t.Fatalf("err %v", err)
	}
	slow := routed(map[string]Dialer{"10.0.0.1:47862": fakeHost(always("x"), 5*time.Second)})
	_, err = Ask(context.Background(), []string{"10.0.0.1"}, request, 200*time.Millisecond, slow, accepting("x"))
	if err == nil || time.Since(started) > 3*time.Second {
		t.Fatalf("a silent host held the search: %v after %v", err, time.Since(started))
	}
}

func TestAHostThatRefusesIsReportedByItsWordsAndNeverAccepted(t *testing.T) {
	dial := routed(map[string]Dialer{"10.0.0.1:47862": fakeHost(func(Request) Answer {
		return Answer{Error: "This PC has no license to share."}
	}, 0)})
	_, err := Ask(context.Background(), []string{"10.0.0.1"}, request, time.Second, dial, func(Answer) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "This PC has no license to share.") {
		t.Fatalf("a refusal was accepted or hidden: %v", err)
	}
}

func TestNoAddressesIsAnErrorAndAHostWithAPortKeepsIt(t *testing.T) {
	if _, err := Ask(context.Background(), nil, request, time.Second, nil, accepting("x")); err == nil {
		t.Fatal("asking nobody succeeded")
	}
	if got := withPort("10.0.0.1"); got != "10.0.0.1:47862" {
		t.Fatalf("bare address: %q", got)
	}
	if got := withPort("127.0.0.1:5000"); got != "127.0.0.1:5000" {
		t.Fatalf("address with a port: %q", got)
	}
}
