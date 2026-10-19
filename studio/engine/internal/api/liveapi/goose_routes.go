package liveapi

import (
	"net/http"

	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/acquisition/svdom"
	"studio-engine/internal/pyjson"
)

func (s *Service) ieds() []*pyjson.Object {
	return pyv.Objects(pyv.GetList(s.Deps.Projects.Current(), "ieds"))
}

// goose is GET /goose: the monitor while running, the SCD watch list while stopped.
func (s *Service) goose(w http.ResponseWriter, r *http.Request) {
	npcap := s.Deps.Network.Npcap()
	if s.running() {
		writeJSON(w, http.StatusOK, pyv.Merge(pyv.O("running", true, "npcap", npcap), s.Live.Goose.Snapshot()))
		return
	}
	watchList := goosedom.WatchList(s.ieds())
	watches := []any{}
	for _, key := range watchList.Keys() {
		watch := pyv.GetObj(watchList, key)
		watch.Set("state", "stopped").Set("problem", "").Set("frames", int64(0))
		watches = append(watches, watch)
	}
	writeJSON(w, http.StatusOK, pyv.O("running", false, "npcap", npcap, "state", "stopped", "unknown", []any{},
		"watches", watches, "engine", s.Live.Engine.Snapshot(), "message", "Press Run to start GOOSE"))
}

// gooseLinks is GET /goose/links: the declared link map joined with the listener's view.
func (s *Service) gooseLinks(w http.ResponseWriter, r *http.Request) {
	running := s.running()
	var watches []*pyjson.Object
	if running {
		watches = s.Live.Goose.Watches()
	}
	project := s.Deps.Projects.Current()
	declarations, err := s.Declarations(project)
	if err != nil {
		writeError(w, err)
		return
	}
	ieds := []*pyjson.Object{}
	for _, ied := range pyv.Objects(pyv.GetList(project, "ieds")) {
		ieds = append(ieds, pyv.O("id", pyv.Get(ied, "id"), "name", pyv.Get(ied, "name"),
			"role", pyv.GetOr(ied, "role", "simulated")))
	}
	writeJSON(w, http.StatusOK, goosedom.WithLive(goosedom.BuildLinkMap(declarations, ieds), watches, running))
}

// mmsAccess is GET /mms/access.
func (s *Service) mmsAccess(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, pyv.O("running", s.running(), "ieds", s.Access.Snapshot()))
}

// sampledValues is GET /sv: the monitor while running, the SCD streams while stopped.
func (s *Service) sampledValues(w http.ResponseWriter, r *http.Request) {
	npcap := s.Deps.Network.Npcap()
	if s.running() {
		writeJSON(w, http.StatusOK, pyv.Merge(pyv.O("running", true, "npcap", npcap), s.Live.SV.Snapshot()))
		return
	}
	network := pyv.GetObj(s.Deps.Projects.Current(), "network")
	nominal := pyv.GetOr(network, "nominal_hz", int64(50))
	supervised, extra := svdom.WatchList(s.ieds(), nominal)
	watches := make([]any, len(supervised))
	for index, watch := range supervised {
		watches[index] = watch
	}
	notSupervised := []any{}
	for _, item := range extra {
		notSupervised = append(notSupervised, pyv.O("key", pyv.Get(item, "key"), "publisher", pyv.Get(item, "publisher")))
	}
	writeJSON(w, http.StatusOK, pyv.O("running", false, "npcap", npcap, "state", "stopped", "unknown", []any{},
		"watches", watches, "not_supervised", notSupervised, "nominal_hz", nominal,
		"message", "Press Run to start listening for Sampled Values"))
}

func (s *Service) sharePreview(w http.ResponseWriter, r *http.Request) {
	preview, err := s.Live.Share.Preview(s.Deps.Projects.Current())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Service) shareApply(w http.ResponseWriter, r *http.Request) {
	request, err := body(r)
	if err == nil {
		var preview *pyjson.Object
		if preview, err = s.Live.Share.Apply(request); err == nil {
			writeJSON(w, http.StatusOK, preview)
			return
		}
	}
	writeError(w, err)
}

func (s *Service) shareChoose(w http.ResponseWriter, r *http.Request) {
	request, err := body(r)
	if err == nil {
		var result any
		if result, err = s.Live.Share.Choose(request); err == nil {
			writeJSON(w, http.StatusOK, result)
			return
		}
	}
	writeError(w, err)
}
