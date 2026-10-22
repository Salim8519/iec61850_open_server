package plant

// TapTransformerKind is the power transformer with an on-load tap changer (ADR-0109). Its HV
// side (north/west terminals, ":a") and LV side (south/east, ":b") are separate sections:
// power passes through it in both directions, but an earthed side does not earth the other side.
const TapTransformerKind = "TAP_TRANSFORMER"

// feedThroughTransformers marks the other side of every transformer live (or maybe live) when
// one side is, until nothing changes, so a chain of transformers passes power on.
func (s *Sections) feedThroughTransformers() {
	for changed := true; changed; {
		changed = false
		for _, node := range s.Nodes {
			if node.Kind != TapTransformerKind {
				continue
			}
			high, low := node.ID+":a", node.ID+":b"
			for _, pair := range [2][2]string{{high, low}, {low, high}} {
				if s.live[s.certain.find(pair[0])] && !s.live[s.certain.find(pair[1])] {
					s.live[s.certain.find(pair[1])] = true
					changed = true
				}
				if s.maybe[s.possible.find(pair[0])] && !s.maybe[s.possible.find(pair[1])] {
					s.maybe[s.possible.find(pair[1])] = true
					changed = true
				}
			}
		}
	}
}
