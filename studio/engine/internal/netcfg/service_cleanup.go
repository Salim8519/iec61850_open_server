package netcfg

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// presence answers "does this journaled change still exist?" reading each Windows list at
// most once per call (change 0089).
type presence struct {
	service  *NetworkService
	unblock  []string
	adapters []Adapter
	rules    map[string]bool
	gateway  map[string]bool
	disabled map[string]bool
	read     map[string]bool
}

func (p *presence) firewall() error {
	if p.read["firewall"] {
		return nil
	}
	gateway, disabled, err := p.service.Firewall(p.unblock)
	if err != nil {
		return err
	}
	p.gateway, p.disabled, p.read["firewall"] = gateway, disabled, true
	return nil
}

func (p *presence) present(item *pyjson.Object) (bool, error) {
	f := fieldsOf(item)
	switch f.text("kind") {
	case "alias":
		if !p.read["adapters"] {
			adapters, err := p.service.Adapters(true)
			if err != nil {
				return false, err
			}
			p.adapters, p.read["adapters"] = adapters, true
		}
		guid, ip := f.text("adapter_guid"), f.text("ip")
		for _, adapter := range p.adapters {
			if adapter.GUID == guid && adapter.ips()[ip] {
				return true, f.err()
			}
		}
		return false, f.err()
	case "gateway_firewall":
		err := p.firewall()
		return err == nil && p.gateway[f.text("name")], firstError(err, f.err())
	case "unblock":
		err := p.firewall()
		return err == nil && p.disabled[f.text("name")], firstError(err, f.err())
	}
	if !p.read["rules"] {
		names, err := p.service.Rules()
		if err != nil {
			return false, err
		}
		p.rules, p.read["rules"] = map[string]bool{}, true
		for _, name := range names {
			p.rules[name] = true
		}
	}
	return p.rules[f.text("name")], f.err()
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Leftovers is service.leftovers: journaled changes that still exist.
func (s *NetworkService) Leftovers() ([]*pyjson.Object, error) {
	outstanding, err := s.Journal.Outstanding()
	if err != nil {
		return nil, err
	}
	check := &presence{service: s, read: map[string]bool{}, unblock: []string{}}
	for _, item := range outstanding {
		if get(item, "kind") == "unblock" {
			check.unblock = append(check.unblock, fieldsOf(item).text("name"))
		}
	}
	return s.Journal.Leftovers(check.present)
}

// CleanupPreview is service.cleanup_preview: what clean-up would remove, fingerprinted.
func (s *NetworkService) CleanupPreview() (*pyjson.Object, error) {
	leftovers, err := s.Leftovers()
	if err != nil {
		return nil, err
	}
	removals := make([]*pyjson.Object, len(leftovers))
	for index, item := range leftovers {
		removals[index] = copyObject(item).Set("action", "remove")
	}
	items, err := withDescriptions(removals, DescribeRemoval)
	if err != nil {
		return nil, err
	}
	digest, err := sortedFingerprint(itemsAny(items))
	if err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("items", itemsAny(items)).Set("fingerprint", digest).
		Set("needs_admin", len(items) > 0), nil
}

// Cleanup is service.cleanup: remove ScadaBench's changes after the user confirmed the list,
// then put the project back in "This PC only" mode.
func (s *NetworkService) Cleanup(fingerprint, confirm any) (*pyjson.Object, error) {
	if !isTrue(confirm) {
		return nil, apperr.New("Confirm the clean-up list first")
	}
	preview, err := s.CleanupPreview()
	if err != nil {
		return nil, err
	}
	digest, _ := preview.Get("fingerprint")
	if fingerprint != digest {
		return nil, conflict("The list of ScadaBench changes is different now. Review it again.", nil)
	}
	raw, _ := preview.Get("items")
	items := objects(raw)
	results := []any{}
	if len(items) > 0 {
		if results, err = s.RunHelper("remove", items, digest.(string)); err != nil {
			return nil, err
		}
	}
	if err := s.Projects.LeaveLabMode(); err != nil {
		return nil, err
	}
	s.clearCache()
	return pyjson.NewObject().Set("results", results).Set("message", "ScadaBench's network changes were removed. "+
		"IEDs are back on 'This PC only' mode."), nil
}
