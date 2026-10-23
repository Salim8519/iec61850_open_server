package contract

// The IED component and GOOSE links (ADR-0112) are Go-only. An IED component is the kind "IED":
// one whole IED on the SLD, chosen with the omittable `ied_id` the point panel already uses.
// `goose_link` is an omittable setting of other equipment: which simulated IED message it sends
// and which values. Documents without them keep their exact bytes.

// iedComponentKind is the kind name; the project package repeats it as IEDComponentKind.
const iedComponentKind = "IED"

// gooseLink names one sendable GOOSE block of one IED and the dataset leaves an equipment sends:
// mode "one" sends `one` (a position, an on/off value or a number); mode "contacts" sends the two
// on/off values `open` and `closed` together. The project package checks the leaves against the
// block's dataset (functional constraint aware) and the equipment kind.
var gooseLink = model("GooseLink", true,
	required("ied_id", text(1, 0, "")),
	required("gocb_ref", text(1, 0, "")),
	defaulted("mode", literal("one", "contacts"), "one"),
	defaulted("one", optional(text(1, 0, "")), nil),
	defaulted("open", optional(text(1, 0, "")), nil),
	defaulted("closed", optional(text(1, 0, "")), nil))
