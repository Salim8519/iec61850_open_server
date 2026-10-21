package licensing

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"studio-engine/internal/licensing/hostkey"
)

// vmRig is a licensed host PC serving on a loopback port, and a virtual machine (a Service with its
// own empty license home) that looks for it. The host speaks the real protocol over real TCP.
type vmRig struct {
	t       *testing.T
	host    *hkHost
	vm      *Service
	vmHome  string
	vmHWID  string
	now     *float64
	dials   atomic.Int32
	address string
	stop    context.CancelFunc
	dial    hostkey.Dialer // nil: the real network
}

const vmHWIDText = "ee"

func newVMRig(t *testing.T) *vmRig {
	t.Helper()
	signer := newHKSigner(t)
	host := newHKHost(t, signer)
	host.install(nil)
	rig := &vmRig{t: t, host: host, vmHWID: strings.Repeat(vmHWIDText, 32), now: host.now}
	server := &hostkey.Server{Responder: host.responder(), Rescan: 10 * time.Millisecond, Logf: t.Logf,
		Listeners: func() []hostkey.Endpoint { return []hostkey.Endpoint{{Addr: "127.0.0.1:0"}} }}
	ctx, cancel := context.WithCancel(context.Background())
	rig.stop = cancel
	done := make(chan struct{})
	go func() { server.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	for deadline := time.Now().Add(3 * time.Second); rig.address == "" && time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if addrs := server.Addrs(); len(addrs) == 1 {
			rig.address = addrs[0]
		}
	}
	if rig.address == "" {
		t.Fatal("the host never started listening")
	}
	rig.vm = rig.newVM()
	return rig
}

// newVM is a computer with its own license home, its own machine ID and a search for the host.
func (r *vmRig) newVM() *Service {
	r.vmHome = hkTempDir(r.t)
	options := VMOptions{Hosts: func() []string { return []string{r.address} }, Timeout: 2 * time.Second, RetryAfter: 30 * time.Second,
		Dial: func(ctx context.Context, address string) (net.Conn, error) {
			r.dials.Add(1)
			if r.dial != nil {
				return r.dial(ctx, address)
			}
			return hostkey.SystemDial(ctx, address)
		}}
	return New(Options{Home: r.vmHome, DataDir: r.vmHome, Enforce: true, Keys: r.host.signer.keys(),
		HWID: func() (string, error) { return r.vmHWID, nil }, Clock: func() float64 { return *r.now },
		Debugger: func() bool { return false }, Logf: r.t.Logf, Registry: newHKRegistry(),
		HostKey: HostKeyOptions{VM: &options}})
}

func (r *vmRig) advance(seconds float64) { *r.now += seconds }

// status asks the VM, waiting for the background search a first unlicensed answer starts.
func (r *vmRig) status() Status {
	r.t.Helper()
	status, err := r.vm.Current(true)
	if err != nil {
		r.t.Fatal(err)
	}
	return status
}

// untilLicensed polls like the browser does until the background search has made the VM licensed.
func (r *vmRig) untilLicensed() Status {
	r.t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if status := r.status(); status.Licensed {
			return status
		}
	}
	r.t.Fatal("the VM was never licensed by its host")
	return Status{}
}

// dynamic is a "host" that reads the VM's request and answers it with respond: it can sign for the
// exact nonce, replay an old proof or lie.
func dynamic(respond func(hostkey.Request) hostkey.Answer) hostkey.Dialer {
	return func(ctx context.Context, address string) (net.Conn, error) {
		client, host := net.Pipe()
		go func() {
			defer host.Close()
			host.SetDeadline(time.Now().Add(3 * time.Second))
			buffer := make([]byte, hostkey.MaxRequestBytes)
			count, err := host.Read(buffer)
			if err != nil {
				return
			}
			var request hostkey.Request
			if json.Unmarshal(buffer[:count], &request) != nil {
				return
			}
			data, _ := json.Marshal(respond(request))
			host.Write(append(data, '\n'))
		}()
		return client, nil
	}
}

// canned is a "host" that sends the same answer whatever it is asked.
func canned(answer hostkey.Answer) hostkey.Dialer {
	return dynamic(func(hostkey.Request) hostkey.Answer { return answer })
}

// attacker is a host with a certificate of its choice (signed by the release key unless said
// otherwise) and a key pair of its own: it signs every request correctly for that key.
type attacker struct {
	private ed25519.PrivateKey
	public  string
}

func newAttacker(t *testing.T) attacker {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return attacker{private: private, public: hostkey.EncodeKey(public)}
}

// serves makes the VM talk to this attacker, whose certificate is the signer's signature over fields.
func (a attacker) serves(rig *vmRig, signer *hkSigner, fields map[string]any) {
	merged := map[string]any{"host_key": a.public}
	for key, value := range fields {
		merged[key] = value
	}
	bundle := signer.sign(rig.t, merged)
	certificate, _ := json.Marshal(map[string]string{"payload": bundle.Payload, "signature": bundle.Signature})
	rig.dial = dynamic(func(request hostkey.Request) hostkey.Answer {
		return hostkey.Answer{Certificate: certificate, Signature: hostkey.Sign(a.private, request)}
	})
}

// answerOf is what the real host answers for request (a genuine proof to replay or change).
func (r *vmRig) answerOf(request hostkey.Request) hostkey.Answer {
	answer, err := r.host.responder().Answer(request)
	if err != nil {
		r.t.Fatal(err)
	}
	return answer
}
