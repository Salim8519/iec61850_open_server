package hostkey

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// Responder answers one proof request, or refuses with a plain-words error (the host's license is
// not valid, the key does not match, ...). It never sees the network.
type Responder interface {
	Answer(request Request) (Answer, error)
}

// Endpoint is one address to serve on. Subnet, when set, is the only network whose peers are
// answered: the virtual switch the adapter belongs to.
type Endpoint struct {
	Addr   string
	Subnet *net.IPNet
}

// Server serves proof requests on the endpoints Listeners names, re-read every Rescan so a
// switch that appears later is served and one that went away is closed.
type Server struct {
	Responder Responder
	Listeners func() []Endpoint
	Limiter   *Limiter
	Rescan    time.Duration // 0 means 30 s
	Logf      func(string, ...any)

	mu     sync.Mutex
	active map[string]*listener
	slots  chan struct{}
}

type listener struct {
	conn     net.Listener
	endpoint Endpoint
}

// maxConnections bounds the requests handled at the same moment.
const maxConnections = 16

func (s *Server) logf(format string, args ...any) {
	if s.Logf != nil {
		s.Logf(format, args...)
		return
	}
	log.Printf(format, args...)
}

// Run serves until ctx ends, then closes every listener.
func (s *Server) Run(ctx context.Context) {
	s.slots = make(chan struct{}, maxConnections)
	rescan := s.Rescan
	if rescan <= 0 {
		rescan = 30 * time.Second
	}
	ticker := time.NewTicker(rescan)
	defer ticker.Stop()
	for {
		s.reconcile(ctx)
		select {
		case <-ctx.Done():
			s.closeAll()
			return
		case <-ticker.C:
		}
	}
}

// Addrs are the addresses being served right now (the real ones, e.g. after port 0).
func (s *Server) Addrs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := []string{}
	for _, item := range s.active {
		found = append(found, item.conn.Addr().String())
	}
	return found
}

func (s *Server) reconcile(ctx context.Context) {
	wanted := map[string]Endpoint{}
	for _, endpoint := range s.Listeners() {
		wanted[endpoint.Addr] = endpoint
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == nil {
		s.active = map[string]*listener{}
	}
	for key, item := range s.active {
		if _, keep := wanted[key]; !keep {
			item.conn.Close()
			delete(s.active, key)
			s.logf("Stopped serving on %s", key)
		}
	}
	for key, endpoint := range wanted {
		if _, running := s.active[key]; running {
			continue
		}
		conn, err := net.Listen("tcp4", endpoint.Addr)
		if err != nil {
			s.logf("Cannot serve on %s: %v", endpoint.Addr, err)
			continue
		}
		item := &listener{conn: conn, endpoint: endpoint}
		s.active[key] = item
		s.logf("Serving virtual machine proofs on %s", conn.Addr())
		go s.accept(ctx, item)
	}
}

func (s *Server) closeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, item := range s.active {
		item.conn.Close()
		delete(s.active, key)
	}
}

func (s *Server) accept(ctx context.Context, item *listener) {
	for {
		conn, err := item.conn.Accept()
		if err != nil {
			return // closed by reconcile or shutdown
		}
		select {
		case s.slots <- struct{}{}:
			go func() {
				defer func() { <-s.slots }()
				s.handle(conn, item.endpoint)
			}()
		default:
			conn.Close() // too many at once
		}
	}
}

// handle answers one request on conn and closes it. Peers outside the switch's subnet are dropped
// without an answer; a peer over its rate limit is told so.
func (s *Server) handle(conn net.Conn, endpoint Endpoint) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	remote, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return
	}
	if peer := net.ParseIP(remote); endpoint.Subnet != nil && (peer == nil || !endpoint.Subnet.Contains(peer)) {
		return
	}
	if s.Limiter != nil && !s.Limiter.Allow(remote) {
		reply(conn, Answer{Error: "Too many requests. Wait a minute."})
		return
	}
	line, err := bufio.NewReaderSize(io.LimitReader(conn, MaxRequestBytes+1), MaxRequestBytes+1).ReadBytes('\n')
	if err != nil || len(line) > MaxRequestBytes {
		return
	}
	var request Request
	if json.Unmarshal(line, &request) != nil {
		reply(conn, Answer{Error: "The request is not understood."})
		return
	}
	if err := request.Validate(); err != nil {
		reply(conn, Answer{Error: err.Error()})
		return
	}
	answer, err := s.Responder.Answer(request)
	if err != nil {
		reply(conn, Answer{Error: err.Error()})
		return
	}
	reply(conn, answer)
}

func reply(conn net.Conn, answer Answer) {
	data, err := json.Marshal(answer)
	if err != nil {
		return
	}
	conn.Write(append(data, '\n'))
}
