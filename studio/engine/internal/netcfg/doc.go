// Package netcfg is the Go port of studio_app/networking (Python, frozen): the adapter list
// (IP Helper through syscall), the lab address plan, the fingerprinted change plan, the
// app-owned journal, the gateway firewall plan, the elevated helper's request validation and
// runner, the reachability probe and the loopback/endpoint checks.
//
// Every value that reaches a file, a hash or a REST body is built as pyjson values so the bytes
// equal what Python wrote (rule names and fingerprints are hashes of Python json.dumps text).
// Everything that touches Windows goes through an interface: AdapterSource, Executor, Elevator,
// Prober and Binder. Tests use fakes; the real implementations change nothing by themselves
// except the Executor and Elevator, which only the network helper flow calls after the user
// confirmed an exact preview.
package netcfg
