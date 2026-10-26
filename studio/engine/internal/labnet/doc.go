// Package labnet is the "This PC lab network" (ADR-0120): one app-owned Windows loopback card,
// "ScadaBench Lab", that holds each simulated IED's own address as a /32, and the small service
// that adds and removes those addresses without a Windows prompt.
//
// The pure part (address rules, the pipe protocol and the request handler) has no Windows calls
// and is tested with fakes. The Windows part creates the card from netloop.inf (SetupAPI), adds
// and removes addresses through the IP Helper API, serves and calls the named pipe, runs as a
// service and pings. Nothing here changes Windows by itself: the card and the service exist only
// after the user's "Create the lab network card" and its UAC approval, and the service accepts
// only add/remove of one unicast IPv4 address on that card.
package labnet
