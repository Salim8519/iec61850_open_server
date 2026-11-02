package runtime

import (
	"errors"
	"fmt"
	"syscall"

	"studio-engine/internal/apperr"
	"studio-engine/internal/portholder"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Windows socket errors of the listen check (ADR-0134).
const (
	wsaAccess          = syscall.Errno(10013) // WSAEACCES: a program holds the port for itself
	wsaAddressInUse    = syscall.Errno(10048) // WSAEADDRINUSE: a program listens on this address
	wsaAddressNotValid = syscall.Errno(10049) // WSAEADDRNOTAVAIL: the address is not usable here
)

// holderOf finds the program on address:port; tests replace it.
var holderOf = portholder.Find

// listenFailure is a failed listen check: the message the user reads and the run-log event
// with the error and the program that holds the port.
type listenFailure struct {
	message *apperr.Error
	event   *pyjson.Object
}

func (f *listenFailure) Error() string { return f.message.Error() }

// Unwrap lets the HTTP layer answer as for any *apperr.Error.
func (f *listenFailure) Unwrap() error { return f.message }

// listenProblem explains why name cannot listen on host:port. The check itself is unchanged; only
// its message says what Windows answered (it said "another program is using it" for every error).
func listenProblem(name, host string, port int64, failure error) *listenFailure {
	detail := osErrorText(failure).Error()
	where := fmt.Sprintf("%s:%d", host, port)
	holder, found := holderOf(host, int(port))
	who := "another program"
	if found {
		who = holder.Label()
	}
	var errno syscall.Errno
	errors.As(failure, &errno)
	message := fmt.Sprintf("%s cannot listen on %s because %s is using it (%s). Close that program or change the port.",
		name, where, who, detail)
	switch {
	case errno == wsaAccess && found && holder.Address == portholder.Any:
		message = fmt.Sprintf("%s cannot listen on %s because %s holds port %d on every address of this PC and "+
			"lets no other program use it (%s). Close that program or give this IED another port.", name, where,
			who, port, detail)
	case errno == wsaAccess:
		message = fmt.Sprintf("%s cannot listen on %s because Windows refused it (%s): %s holds port %d for itself, "+
			"or security software blocks it. Close that program or give this IED another port.", name, where,
			detail, who, port)
	case errno == wsaAddressNotValid:
		message = fmt.Sprintf("%s cannot listen on %s because this address is not usable on this PC right now (%s). "+
			"If it is on a network card, check that the card is enabled and its cable is plugged in.", name, where, detail)
	}
	event := pyval.Obj("ied", name, "endpoint", where, "error", detail)
	if found {
		event.Set("holder_pid", int64(holder.PID)).Set("holder_program", holder.Program).
			Set("holder_address", holder.Address)
	}
	return &listenFailure{message: apperr.New(message), event: event}
}
