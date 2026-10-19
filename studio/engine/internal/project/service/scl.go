package service

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project"
	"studio-engine/internal/project/netset"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// AddSCL is ProjectService.add_scl: import (outside the lock), name and capacity checks,
// default IED settings, loopback addresses, save and change.
func (s *Service) AddSCL(source []byte, filename string, selection scl.Selection) ([]*pyjson.Object, error) {
	models, err := s.config.Importer.Import(source, filename, selection)
	if err != nil {
		return nil, err
	}
	s.mutate.Lock()
	defer s.mutate.Unlock()
	next := draft(s.current.Load().project)
	ieds := iedList(next)
	names := map[string]bool{}
	for _, ied := range ieds {
		if name, ok := ied.Get("name"); ok {
			if text, isText := name.(string); isText {
				names[text] = true
			}
		}
	}
	for _, model := range models {
		if name, _ := model.Get("name"); names[name.(string)] {
			return nil, apperr.New("An IED with this name already exists; rename before importing")
		}
	}
	if len(models)+len(names) > 20 {
		return nil, apperr.New("This release supports up to 20 IEDs per project")
	}
	for _, model := range models {
		defaults, err := netset.ValidateIEDSettings(pyjson.NewObject())
		if err != nil {
			return nil, err
		}
		for _, key := range defaults.Keys() {
			value, _ := defaults.Get(key)
			model.Set(key, value)
		}
	}
	ieds = append(ieds, models...)
	if err := project.AssignEndpoints(ieds); err != nil {
		return nil, err
	}
	next.Set("ieds", objects(ieds))
	if err := s.commit(next, s.current.Load().notice, true, nil); err != nil {
		return nil, err
	}
	return models, nil
}

// AddTemplate is ProjectService.add_template(name).
func (s *Service) AddTemplate(name string) ([]*pyjson.Object, error) {
	source, err := scl.MakeTemplate(name)
	if err != nil {
		return nil, err
	}
	return s.AddSCL(source, name+".cid", scl.SelectAll)
}

// FindIED is ProjectService.find_ied: StudioError "Unknown IED" (code not_found).
func (s *Service) FindIED(key string) (*pyjson.Object, error) {
	return project.FindIED(s.current.Load().project, key)
}
