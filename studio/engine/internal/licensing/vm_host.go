package licensing

import (
	"context"
	"errors"
	"sync"

	"studio-engine/internal/licensing/hostkey"
)

// ModeVMHost is the status mode of a virtual machine that is licensed through its host.
const ModeVMHost = "vm-host"

// vmProof is a host's confirmation this process verified: whose license, which host key, when.
type vmProof struct {
	at        int64
	licenseID any
	hostKey   string
}

// vmHostState is the VM side's memory: the proof, and whether a search is running or must wait.
type vmHostState struct {
	refresh sync.Mutex // one search at a time: two never write the proof file together
	mu      sync.Mutex
	proof   *vmProof
	running bool
	nextTry float64
}

// acceptHostAnswer is every rule of the contract for an answer a VM gets: the certificate verifies
// with the release keys, is a lifetime license that is not cancelled here and names a host key, and
// the signature verifies for this request (this nonce and this VM's machine ID).
func (s *Service) acceptHostAnswer(answer hostkey.Answer, request hostkey.Request) (*vmProof, error) {
	bundle, err := ParseBundle(string(answer.Certificate))
	if err != nil {
		return nil, err
	}
	payload, err := VerifyBundle(bundle, s.options.Keys)
	if err != nil {
		return nil, err
	}
	version, damaged, err := payloadInt(payload, "version", true)
	if err != nil || damaged || version != 1 {
		return nil, errors.New(Unsupported)
	}
	if plan, _ := payload.Get("type"); plan != any("lifetime") {
		return nil, errors.New(HostNotLifetimeText)
	}
	now := s.now()
	issued, damaged, err := payloadInt(payload, "issued_at", false)
	if err != nil || damaged {
		return nil, errors.New(Damaged)
	}
	expires, _, _ := payloadInt(payload, "expires_at", true)
	switch {
	case now+ClockToleranceSeconds < max(s.anchor.MaxSeen(), issued):
		return nil, errors.New(ClockRollback)
	case expires > 0 && now >= expires:
		return nil, errors.New(LicenseExpired)
	}
	licenseID, _ := payload.Get("license_id")
	if s.revoked.Contains(licenseID) {
		return nil, errors.New(RevokedText)
	}
	hostText := certHostKey(payload)
	public, err := hostkey.DecodeKey(hostText)
	if err != nil {
		return nil, errors.New("the host's license has no virtual machine key")
	}
	if !hostkey.Verify(public, request, answer.Signature) {
		return nil, errors.New("the host's signature does not match this request")
	}
	return &vmProof{at: now, licenseID: licenseID, hostKey: hostText}, nil
}

// RefreshVMHost asks the hosts near this computer for a proof now (the same search a background
// start makes). On success the proof is kept in memory, written for the native workers as
// vm_proof.json, and the status is re-read. It returns the reason nobody confirmed otherwise.
func (s *Service) RefreshVMHost() error {
	vm := s.options.HostKey.VM
	if vm == nil {
		return errors.New("virtual machines are not offered a host here")
	}
	s.vm.refresh.Lock()
	err := s.askHost(vm)
	s.vm.refresh.Unlock()
	s.vm.mu.Lock()
	s.vm.running = false
	retry := vm.RetryAfter
	if retry <= 0 {
		retry = vmDefaultRetryAfter
	}
	s.vm.nextTry = 0
	if err != nil {
		s.vm.nextTry = s.options.Clock() + retry.Seconds()
	}
	s.vm.mu.Unlock()
	if err == nil {
		s.invalidate()
	}
	return err
}

func (s *Service) askHost(vm *VMOptions) error {
	hwid, err := s.HWID()
	if err != nil {
		return err
	}
	request := hostkey.Request{Nonce: s.options.Nonce(), VMHWID: hwid}
	hosts := hostsOf(vm)
	timeout := vm.Timeout
	if timeout <= 0 {
		timeout = vmDefaultTimeout
	}
	found, err := hostkey.Ask(context.Background(), hosts, request, timeout, vm.Dial, func(answer hostkey.Answer) error {
		_, err := s.acceptHostAnswer(answer, request)
		return err
	})
	if err != nil {
		return err
	}
	accepted, err := s.acceptHostAnswer(found.Answer, request) // the proof of the answer that won
	if err != nil {
		return err
	}
	if err := s.writeVMProof(found.Answer, request, accepted.at); err != nil {
		return err
	}
	s.vm.mu.Lock()
	s.vm.proof = accepted
	s.vm.mu.Unlock()
	return nil
}

func hostsOf(vm *VMOptions) []string {
	if vm.Hosts != nil {
		return vm.Hosts()
	}
	adapters, err := hostkey.SystemAdapters()
	if err != nil {
		return nil
	}
	return hostkey.CandidateHosts(adapters)
}
