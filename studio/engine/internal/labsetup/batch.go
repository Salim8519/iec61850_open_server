package labsetup

import "studio-engine/internal/labnet"

// wanted is one address to put on the card and whose it is (for the journal).
type wanted struct{ address, name string }

// call sends one request to the service and remembers the protocol version it answers with.
func (s *Service) call(request labnet.Request) (labnet.Reply, error) {
	reply, err := s.Windows.Call(request)
	if err == nil && reply.Version != "" {
		s.version.Store(reply.Version)
	}
	return reply, err
}

// serviceVersion is the protocol of the service's last answer ("" before one); reading it costs
// no pipe call.
func (s *Service) serviceVersion() string {
	version, _ := s.version.Load().(string)
	return version
}

// addAll puts addresses on the card in one request when the service takes several (ADR-0128),
// else one request each as protocol 1 needs. Every address keeps its own journal record.
func (s *Service) addAll(items []wanted, card string) {
	if len(items) < 2 || !labnet.Batched(s.serviceVersion()) {
		for _, item := range items {
			s.add(item.address, item.name, card)
		}
		return
	}
	addresses := make([]string, len(items))
	for index, item := range items {
		addresses[index] = item.address
	}
	reply, err := s.call(labnet.Request{Op: labnet.OpAdd, IPs: addresses})
	if err != nil {
		return
	}
	for _, item := range items {
		reason := reply.Failed[item.address]
		if reason == "" && contains(reply.Addresses, item.address) {
			s.journal("applied", "lab_address", item.address, item.name, card, "")
			continue
		}
		if reason == "" {
			reason = reply.Message
		}
		s.journal("failed", "lab_address", item.address, item.name, card, reason)
	}
}

// removeAll takes addresses off the card, in one request when the service takes several.
func (s *Service) removeAll(addresses []string, card string) {
	if len(addresses) < 2 || !labnet.Batched(s.serviceVersion()) {
		for _, address := range addresses {
			s.remove(address, "", card)
		}
		return
	}
	reply, err := s.call(labnet.Request{Op: labnet.OpRemove, IPs: addresses})
	if err != nil {
		return
	}
	for _, address := range addresses {
		event, reason := "removed", reply.Failed[address]
		if reason != "" || (!reply.OK && reply.Failed == nil) {
			event = "failed"
		}
		if event == "failed" && reason == "" {
			reason = reply.Message
		}
		s.journal(event, "lab_address", address, "", card, reason)
	}
}
