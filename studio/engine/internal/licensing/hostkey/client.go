package hostkey

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

// Found is the host that gave an answer the caller accepted.
type Found struct {
	Host   string
	Answer Answer
}

// Dialer opens a connection; tests pass a fake, the app uses the system dialer.
type Dialer func(ctx context.Context, address string) (net.Conn, error)

// SystemDial is a plain TCP dial.
func SystemDial(ctx context.Context, address string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "tcp4", address)
}

// Ask sends request to every host at the same moment and returns the first answer that accept
// approves (it checks the certificate and the signature). A host is a bare IPv4 address (the proof
// port is added) or "address:port". timeout bounds the whole search, so a VM with no host nearby
// is never kept waiting. dial nil means SystemDial.
func Ask(ctx context.Context, hosts []string, request Request, timeout time.Duration, dial Dialer,
	accept func(Answer) error) (Found, error) {
	if dial == nil {
		dial = SystemDial
	}
	if len(hosts) == 0 {
		return Found{}, errors.New("this computer has no network address where a host could answer")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	type result struct {
		found Found
		err   error
	}
	results := make(chan result, len(hosts))
	for _, host := range hosts {
		go func(host string) {
			answer, err := askOne(ctx, withPort(host), request, dial)
			if err == nil {
				err = accept(answer)
			}
			results <- result{Found{Host: host, Answer: answer}, err}
		}(host)
	}
	var last error
	for range hosts {
		got := <-results
		if got.err == nil {
			return got.found, nil
		}
		last = got.err
	}
	return Found{}, fmt.Errorf("no host confirmed this license (%d addresses asked): %w", len(hosts), last)
}

// withPort adds the proof port to a bare address.
func withPort(host string) string {
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	return net.JoinHostPort(host, strconv.Itoa(Port))
}

func askOne(ctx context.Context, address string, request Request, dial Dialer) (Answer, error) {
	conn, err := dial(ctx, address)
	if err != nil {
		return Answer{}, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	data, err := json.Marshal(request)
	if err != nil {
		return Answer{}, err
	}
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return Answer{}, err
	}
	line, err := bufio.NewReader(io.LimitReader(conn, MaxAnswerBytes+1)).ReadBytes('\n')
	if err != nil || len(line) > MaxAnswerBytes {
		return Answer{}, errors.New("no answer")
	}
	var answer Answer
	if err := json.Unmarshal(line, &answer); err != nil {
		return Answer{}, errors.New("the answer is not understood")
	}
	if answer.Error != "" {
		return Answer{}, errors.New(answer.Error)
	}
	return answer, nil
}
