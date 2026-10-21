package licensing

import (
	"studio-engine/internal/licensing/hostkey"
	"studio-engine/internal/pyjson"
)

// evaluate is the status of this computer: its own license (evaluateOwn) and, when it has none that
// is valid, the proof of a licensed host it runs on (docs/architecture/license-binding-contract.md,
// "VM host key"). A proof never replaces a valid license of the computer's own, and it never gets
// past the integrity check: a swapped worker or an attached debugger is unlicensed whatever a host says.
func (s *Service) evaluate() (Status, error) {
	status, err := s.evaluateOwn()
	if err != nil || status.Licensed || !s.options.Enforce || s.options.HostKey.VM == nil {
		return status, err
	}
	if s.integrityProblem() != "" {
		return status, nil
	}
	return s.vmHostStatus(status), nil
}

func (s *Service) integrityProblem() string {
	folder := s.options.NativeFolder
	if folder == "" {
		folder = "."
	}
	return IntegrityProblem(folder, s.options.ExpectedFiles, s.options.Debugger)
}

// vmHostStatus is licensed while a verified proof younger than 24 hours exists. It never waits for
// the network: a missing or old proof starts one background search and the status stays as it was.
func (s *Service) vmHostStatus(own Status) Status {
	now := s.now()
	s.vm.mu.Lock()
	proof := s.vm.proof
	s.vm.mu.Unlock()
	if proof != nil && now >= proof.at-ClockToleranceSeconds && now-proof.at < vmProofMaxAge {
		if now-proof.at >= vmRefreshAfter {
			s.kickVMHost()
		}
		return Status{Licensed: true, Mode: ModeVMHost, Plan: "lifetime", LicenseID: proof.licenseID}
	}
	s.kickVMHost()
	return own
}

// kickVMHost starts one background search, but not while one runs and not right after a failure.
func (s *Service) kickVMHost() {
	s.vm.mu.Lock()
	if s.vm.running || s.options.Clock() < s.vm.nextTry {
		s.vm.mu.Unlock()
		return
	}
	s.vm.running = true
	s.vm.mu.Unlock()
	go func() {
		if err := s.RefreshVMHost(); err != nil {
			s.options.Logf("No host confirmed this virtual machine: %v", err)
		}
	}()
}

// statusObject is Status.Object() plus what only a virtual machine's status carries: the masked
// host key (Go engine only, absent from every other answer).
func (s *Service) statusObject(current Status) *pyjson.Object {
	object := current.Object()
	if current.Mode != ModeVMHost {
		return object
	}
	s.vm.mu.Lock()
	proof := s.vm.proof
	s.vm.mu.Unlock()
	if proof != nil {
		object.Set("host_key", hostkey.MaskKey(proof.hostKey))
	}
	return object
}
