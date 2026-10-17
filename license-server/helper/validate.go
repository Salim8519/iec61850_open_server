package main

import "regexp"

// hwidPattern is the Machine ID format the Studio app and the server use.
var hwidPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func validHWID(hwid string) bool { return hwidPattern.MatchString(hwid) }
