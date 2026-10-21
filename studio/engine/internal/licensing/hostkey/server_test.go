package hostkey

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// stubResponder answers every request with a fixed answer or refusal and remembers what it saw.
type stubResponder struct {
	mu     sync.Mutex
	seen   []Request
	answer Answer
	err    error
}

func (s *stubResponder) Answer(request Request) (Answer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, request)
	return s.answer, s.err
}

func (s *stubResponder) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.seen)
}

// serve starts a server on a loopback address chosen by the system and returns it with its address.
func serve(t *testing.T, responder Responder, limiter *Limiter, subnet *net.IPNet) (*Server, string) {
	t.Helper()
	server := &Server{Responder: responder, Limiter: limiter, Rescan: 20 * time.Millisecond, Logf: t.Logf,
		Listeners: func() []Endpoint { return []Endpoint{{Addr: "127.0.0.1:0", Subnet: subnet}} }}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { server.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if addrs := server.Addrs(); len(addrs) == 1 {
			return server, addrs[0]
		}
	}
	t.Fatal("the server never started listening")
	return nil, ""
}

// ask sends one raw line and reads one answer line ("" when the server closed without one).
func ask(t *testing.T, address, line string) string {
	t.Helper()
	conn, err := net.DialTimeout("tcp4", address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write([]byte(line)); err != nil {
		return ""
	}
	answer, _ := bufio.NewReader(conn).ReadString('\n')
	return answer
}

func goodLine() string {
	data, _ := json.Marshal(request)
	return string(data) + "\n"
}

func TestAValidRequestGetsTheResponderAnswerAsOneJSONLine(t *testing.T) {
	responder := &stubResponder{answer: Answer{Certificate: json.RawMessage(`{"payload":"p","signature":"s"}`), Signature: "sig"}}
	_, address := serve(t, responder, nil, nil)
	reply := ask(t, address, goodLine())
	var got Answer
	if err := json.Unmarshal([]byte(reply), &got); err != nil || got.Signature != "sig" || got.Error != "" ||
		string(got.Certificate) != `{"payload":"p","signature":"s"}` || strings.Count(reply, "\n") != 1 {
		t.Fatalf("reply %q (%v)", reply, err)
	}
	if responder.seen[0] != request {
		t.Fatalf("the responder saw %+v", responder.seen[0])
	}
}

func TestARefusingHostSaysSoInPlainWordsAndRevealsNothingElse(t *testing.T) {
	responder := &stubResponder{err: errors.New("This PC has no license to share.")}
	_, address := serve(t, responder, nil, nil)
	var got Answer
	json.Unmarshal([]byte(ask(t, address, goodLine())), &got)
	if got.Error != "This PC has no license to share." || got.Signature != "" || len(got.Certificate) != 0 {
		t.Fatalf("a refusal carried more than its words: %+v", got)
	}
}

func TestBadRequestsNeverReachTheResponder(t *testing.T) {
	responder := &stubResponder{}
	_, address := serve(t, responder, nil, nil)
	bad := map[string]string{
		"not json":        "hello\n",
		"short nonce":     `{"nonce":"ab","vm_hwid":"` + request.VMHWID + `"}` + "\n",
		"short machine":   `{"nonce":"` + request.Nonce + `","vm_hwid":"cd"}` + "\n",
		"upper-case id":   `{"nonce":"` + request.Nonce + `","vm_hwid":"` + strings.ToUpper(request.VMHWID) + `"}` + "\n",
		"empty object":    "{}\n",
		"an array":        "[]\n",
		"nothing at all":  "\n",
		"no newline end":  `{"nonce":"` + request.Nonce + `"}`,
		"a huge request":  strings.Repeat("x", MaxRequestBytes+10) + "\n",
		"a huge no-eol":   strings.Repeat("y", MaxRequestBytes*3),
		"binary garbage":  "\x00\x01\x02\xff\xfe\n",
		"a second object": goodLine() + goodLine(),
	}
	for name, line := range bad {
		reply := ask(t, address, line)
		if name == "a second object" {
			continue // the first line is a valid request: it is answered, the rest is ignored
		}
		if strings.Contains(reply, `"signature"`) || strings.Contains(reply, `"certificate"`) {
			t.Errorf("%s was answered with a proof: %q", name, reply)
		}
	}
	for _, seen := range responder.seen {
		if seen != request {
			t.Fatalf("the responder saw a request that was not valid: %+v", seen)
		}
	}
}

func TestAPeerOverTheRateLimitIsToldToWaitAndIsNotAnswered(t *testing.T) {
	responder := &stubResponder{answer: Answer{Signature: "sig"}}
	limiter := &Limiter{PerHost: 2, Total: 100, Window: time.Minute}
	_, address := serve(t, responder, limiter, nil)
	var replies []Answer
	for index := 0; index < 4; index++ {
		var got Answer
		json.Unmarshal([]byte(ask(t, address, goodLine())), &got)
		replies = append(replies, got)
	}
	if replies[0].Signature != "sig" || replies[1].Signature != "sig" {
		t.Fatalf("the first two requests were not answered: %+v", replies)
	}
	if replies[2].Signature != "" || !strings.Contains(replies[2].Error, "Too many requests") || responder.count() != 2 {
		t.Fatalf("the third request was answered: %+v (%d answered)", replies[2], responder.count())
	}
}

func TestAPeerOutsideTheSwitchSubnetGetsNoAnswerAtAll(t *testing.T) {
	_, elsewhere, _ := net.ParseCIDR("192.168.77.0/24")
	responder := &stubResponder{answer: Answer{Signature: "sig"}}
	_, address := serve(t, responder, nil, elsewhere) // the client comes from 127.0.0.1: outside
	if reply := ask(t, address, goodLine()); reply != "" || responder.count() != 0 {
		t.Fatalf("a peer outside the subnet was answered: %q (%d)", reply, responder.count())
	}
	_, here, _ := net.ParseCIDR("127.0.0.0/8")
	responder2 := &stubResponder{answer: Answer{Signature: "sig"}}
	_, address2 := serve(t, responder2, nil, here)
	if reply := ask(t, address2, goodLine()); !strings.Contains(reply, "sig") {
		t.Fatalf("a peer inside the subnet was not answered: %q", reply)
	}
}

func TestListenersFollowTheAdaptersThatAppearAndDisappear(t *testing.T) {
	var mu sync.Mutex
	wanted := []Endpoint{}
	server := &Server{Responder: &stubResponder{}, Rescan: 10 * time.Millisecond, Logf: t.Logf,
		Listeners: func() []Endpoint { mu.Lock(); defer mu.Unlock(); return append([]Endpoint(nil), wanted...) }}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { server.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	waitFor := func(count int) []string {
		for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
			if addrs := server.Addrs(); len(addrs) == count {
				return addrs
			}
		}
		t.Fatalf("the server never reached %d listeners (has %v)", count, server.Addrs())
		return nil
	}
	waitFor(0) // no virtual switch yet: nothing is open
	mu.Lock()
	wanted = []Endpoint{{Addr: "127.0.0.1:0"}}
	mu.Unlock()
	addrs := waitFor(1)
	if reply := ask(t, addrs[0], goodLine()); !strings.Contains(reply, "{") {
		t.Fatalf("the new listener does not answer: %q", reply)
	}
	mu.Lock()
	wanted = nil // the switch went away, or the license ended
	mu.Unlock()
	waitFor(0)
	if conn, err := net.DialTimeout("tcp4", addrs[0], 300*time.Millisecond); err == nil {
		conn.Close()
		t.Fatal("the closed listener still accepts connections")
	}
}

func TestEndingTheServerClosesEveryListener(t *testing.T) {
	server := &Server{Responder: &stubResponder{}, Rescan: 10 * time.Millisecond, Logf: t.Logf,
		Listeners: func() []Endpoint { return []Endpoint{{Addr: "127.0.0.1:0"}} }}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { server.Run(ctx); close(done) }()
	var address string
	for deadline := time.Now().Add(3 * time.Second); address == "" && time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if addrs := server.Addrs(); len(addrs) == 1 {
			address = addrs[0]
		}
	}
	if address == "" {
		t.Fatal("the server never started")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("the server did not end")
	}
	if conn, err := net.DialTimeout("tcp4", address, 300*time.Millisecond); err == nil {
		conn.Close()
		t.Fatal("a listener survived the shutdown")
	}
}
