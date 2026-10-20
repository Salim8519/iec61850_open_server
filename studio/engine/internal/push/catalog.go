package push

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"studio-engine/internal/api"
)

// Change names the services report through their OnChange hooks (gateway/service,
// tools/explorer, tools/scan). They are also topic triggers.
const (
	ChangeGatewayStatus   = "gateway.status"
	ChangeGatewayPoints   = "gateway.points"
	ChangeGatewayCommands = "gateway.commands"
	ChangeGatewayTraffic  = "gateway.traffic"
	ChangeMasterStatus    = "master.status"
	ChangeMasterPoints    = "master.points"
	ChangeMasterCommands  = "master.commands"
	ChangeMasterTraffic   = "master.traffic"
	ChangeExplorer        = "explorer"
	ChangeScan            = "scan"
)

// CatalogOptions are the live conditions some topics need for their periodic re-read.
type CatalogOptions struct {
	// Running reports a running simulation (runtime state != "stopped"): counters and
	// staleness are re-read every second only then.
	Running func() bool
	// Scanning reports a running Find IEDs scan (its "seconds" counter).
	Scanning func() bool
}

// Spec is one catalog entry: the topic and the GET pattern whose handler produces it.
type Spec struct {
	Pattern string
	Topic   Topic
}

const second = time.Second

// PointsGap is the coalescing interval of the gateway and master point tables.
const PointsGap = time.Second

// Specs lists every topic of the app with its polled GET route.
func Specs(o CatalogOptions) []Spec {
	running, scanning := o.Running, o.Scanning
	whole := func(name, pattern string, triggers ...string) Spec {
		return Spec{Pattern: "/api/v1" + pattern, Topic: Topic{Name: name, Triggers: triggers}}
	}
	ticking := func(spec Spec, while func() bool) Spec {
		if while != nil { // without the condition there is no periodic re-read at all
			spec.Topic.Every, spec.Topic.While = second, while
		}
		return spec
	}
	keyed := func(spec Spec, kind Kind, list, key string) Spec {
		spec.Topic.Kind, spec.Topic.List, spec.Topic.Key = kind, list, key
		return spec
	}
	health := ticking(whole("health", "/health", "live_*", "goose_*", "sv_*", "mms_access"), running)
	health.Topic.Ignore, health.Topic.Gap, health.Topic.Values = "checked_at", 250*time.Millisecond, ModeRef
	panel := whole("test.panel", "/project/test-panel", "source*", "pulse_*", "scenario_*", "live_*")
	panel.Topic.Values = AnyValue
	points := keyed(whole("gateway.points", "/tools/gateway/points", ChangeGatewayPoints, ChangeGatewayStatus,
		ChangeMasterPoints), Keyed, "points", "id")
	// A 5000-point table body costs about 22 ms to build (BenchmarkTable5000): the point
	// tables re-read at most once a second, never more than the 1 s poll they replace.
	points.Topic.Params, points.Topic.Values, points.Topic.Gap = []string{"protocol"}, AnyValue, PointsGap
	masterPoints := keyed(whole("master.points", "/tools/master/points", ChangeMasterPoints, ChangeMasterStatus),
		Keyed, "points", "key")
	masterPoints.Topic.Gap = PointsGap
	activity := whole("activity", "/activity", ChangeGatewayStatus, ChangeMasterStatus, ChangeExplorer, ChangeScan,
		"scenario_*", "gateway_*", "master_*", "explorer_*")
	activity.Topic.Gap = 250 * time.Millisecond
	return []Spec{
		health,
		ticking(whole("goose.links", "/goose/links", "goose_*"), running),
		ticking(whole("goose.overview", "/goose", "goose_*"), running),
		whole("goose.share", "/goose/share", "goose_share", "goose_conflict", "goose_uplink"),
		ticking(whole("sv", "/sv", "sv_*"), running),
		ticking(whole("live.sessions", "/live", "live_*", "goose_status"), running),
		whole("mms.access", "/mms/access", "mms_access"),
		panel,
		whole("scenario.runs", "/scenarios/runs", "scenario_*"),
		activity,
		whole("gateway.status", "/tools/gateway", ChangeGatewayStatus, "gateway_*"),
		whole("gateway.config", "/tools/gateway/config", ChangeGatewayStatus),
		points,
		keyed(whole("gateway.commands", "/tools/gateway/commands", ChangeGatewayCommands), Keyed, "entries", "seq"),
		keyed(whole("gateway.traffic", "/tools/gateway/traffic", ChangeGatewayTraffic), Keyed, "entries", "seq"),
		whole("master.status", "/tools/master", ChangeMasterStatus, "master_*"),
		whole("master.config", "/tools/master/config", ChangeMasterStatus),
		masterPoints,
		keyed(whole("master.commands", "/tools/master/commands", ChangeMasterCommands), Keyed, "entries", "seq"),
		keyed(whole("master.traffic", "/tools/master/traffic", ChangeMasterTraffic), Keyed, "entries", "seq"),
		whole("explorer", "/tools/explorer", ChangeExplorer, "explorer_*"),
		ticking(whole("scan", "/tools/scan", ChangeScan), scanning),
	}
}

// Catalog binds the specs to the composed routes; missing lists the patterns no area
// serves (those topics are left out, so their views keep polling).
func Catalog(index RouteIndex, o CatalogOptions) (topics []*Topic, missing []string) {
	for _, spec := range Specs(o) {
		get, found := index.Getter(spec.Pattern)
		if !found {
			missing = append(missing, spec.Pattern)
			continue
		}
		topic := spec.Topic
		topic.Get, topic.Path = get, strings.TrimPrefix(spec.Pattern, "/api/v1")
		topics = append(topics, &topic)
	}
	return topics, missing
}

// Coverage compares the catalog with one area's routes: the topics it serves, topic
// patterns it serves but not as a polled GET, and its polled GET routes without a topic
// (each of those is a view that would keep polling on the Go engine).
func Coverage(routes []api.Route) (served, notPolled, uncovered []string) {
	specs := map[string]string{}
	for _, spec := range Specs(CatalogOptions{}) {
		specs[spec.Pattern] = spec.Topic.Name
	}
	for _, route := range routes {
		name, isTopic := specs[route.Pattern]
		switch {
		case isTopic && route.Method == http.MethodGet && route.Polled:
			served = append(served, name)
		case isTopic && route.Method == http.MethodGet:
			notPolled = append(notPolled, name)
		case route.Method == http.MethodGet && route.Polled:
			uncovered = append(uncovered, route.Pattern)
		}
	}
	sort.Strings(served)
	return served, notPolled, uncovered
}

// ModeRef selects the Mod, Beh and LPHD Sim values health shows (and IED-wide changes).
func ModeRef(_ string, ref string) bool {
	return ref == "" || strings.HasSuffix(ref, ".Mod.stVal") || strings.HasSuffix(ref, ".Beh.stVal") ||
		strings.HasSuffix(ref, ".Sim.stVal")
}
