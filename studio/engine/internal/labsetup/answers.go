package labsetup

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

// answerLife is how long an ARP answer is trusted; a real card that changes networks forgets
// its answers at once (the source address is part of the key).
const answerLife = 60 * time.Second

// askParallel is how many ARP questions run at once (each waits up to about 3 s for silence).
const askParallel = 32

type answer struct {
	used bool
	at   time.Time
}

// Asker remembers ARP answers and asks the missing ones in the background, so the page's status
// never waits for the network.
type Asker struct {
	Ask   func(address, source string) bool
	Clock func() time.Time

	mu      sync.Mutex
	known   map[[2]string]answer
	pending map[[2]string]bool
}

// NewAsker returns an Asker over ask (labnet.AskOn in the app).
func NewAsker(ask func(address, source string) bool) *Asker {
	return &Asker{Ask: ask, Clock: time.Now, known: map[[2]string]answer{}, pending: map[[2]string]bool{}}
}

// Answer is (answered, known) for one pair.
func (a *Asker) Answer(address, source string) (bool, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	found, ok := a.known[[2]string{address, source}]
	if !ok || a.Clock().Sub(found.at) > answerLife {
		return false, false
	}
	return found.used, true
}

// Start asks every pair not known yet; it returns at once.
func (a *Asker) Start(pairs [][2]string) {
	a.mu.Lock()
	todo := [][2]string{}
	for _, pair := range pairs {
		found, ok := a.known[pair]
		if a.pending[pair] || (ok && a.Clock().Sub(found.at) <= answerLife) {
			continue
		}
		a.pending[pair] = true
		todo = append(todo, pair)
	}
	a.mu.Unlock()
	if len(todo) > 0 {
		go a.run(todo)
	}
}

// Wait asks the pairs and returns when all are answered (Use and Change wait for the truth).
func (a *Asker) Wait(pairs [][2]string) {
	a.Start(pairs)
	for deadline := a.Clock().Add(10 * time.Second); a.Clock().Before(deadline); time.Sleep(50 * time.Millisecond) {
		missing := false
		for _, pair := range pairs {
			if _, known := a.Answer(pair[0], pair[1]); !known {
				missing = true
				break
			}
		}
		if !missing {
			return
		}
	}
}

func (a *Asker) run(pairs [][2]string) {
	slots := make(chan struct{}, askParallel)
	var group sync.WaitGroup
	for _, pair := range pairs {
		group.Add(1)
		slots <- struct{}{}
		go func(pair [2]string) {
			defer group.Done()
			used := a.Ask(pair[0], pair[1])
			a.mu.Lock()
			a.known[pair] = answer{used: used, at: a.Clock()}
			delete(a.pending, pair)
			a.mu.Unlock()
			<-slots
		}(pair)
	}
	group.Wait()
}

func itoa(value int) string { return strconv.Itoa(value) }

func split(network string) (string, int) {
	ip, prefix, _ := strings.Cut(network, "/")
	value, _ := strconv.Atoi(prefix)
	return ip, value
}
