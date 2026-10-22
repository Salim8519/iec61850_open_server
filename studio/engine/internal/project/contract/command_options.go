package contract

// commandOptions are the command options a breaker, disconnector or earth switch remembers
// (ADR-0108): the test, interlock-check and synchro-check bits and the originator (orCat 0 to 8,
// orIdent up to 64 characters). The field is omittable: it is saved only after options were
// sent once, so documents of nodes that never sent options keep their exact bytes.
var commandOptions = model("CommandOptions", true,
	defaulted("test", boolean, false),
	defaulted("interlock_check", boolean, false),
	defaulted("synchro_check", boolean, false),
	defaulted("orcat", integer(true, 0, 8), int64(0)),
	defaulted("orident", text(0, 64, ""), ""))
