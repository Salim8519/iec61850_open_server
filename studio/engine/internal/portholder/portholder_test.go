package portholder

import (
	"encoding/binary"
	"testing"
)

// row is one MIB_TCPROW_OWNER_PID as Windows writes it.
func row(address [4]byte, port int, pid uint32) []byte {
	data := make([]byte, 24)
	binary.LittleEndian.PutUint32(data, 2) // LISTEN
	copy(data[4:8], address[:])
	data[8], data[9] = byte(port>>8), byte(port)
	binary.LittleEndian.PutUint32(data[20:], pid)
	return data
}

func TestParseTableAndPick(t *testing.T) {
	table := []byte{3, 0, 0, 0}
	table = append(table, row([4]byte{0, 0, 0, 0}, 102, 4120)...)
	table = append(table, row([4]byte{127, 50, 0, 2}, 102, 77)...)
	table = append(table, row([4]byte{127, 0, 0, 1}, 8080, 9)...)
	rows := parseTable(table)
	if len(rows) != 3 || rows[1] != (listener{"127.50.0.2", 102, 77}) {
		t.Fatalf("rows %+v", rows)
	}
	if got, ok := pick(rows, "127.50.0.2", 102); !ok || got.pid != 77 {
		t.Fatalf("exact address: %+v %v", got, ok)
	}
	if got, ok := pick(rows, "127.50.0.3", 102); !ok || got.pid != 4120 || got.address != Any {
		t.Fatalf("every-address holder: %+v %v", got, ok)
	}
	if _, ok := pick(rows, "127.0.0.1", 102); !ok {
		t.Fatal("the every-address holder holds 127.0.0.1:102 too")
	}
	if _, ok := pick(rows, "127.50.0.2", 103); ok {
		t.Fatal("found a holder on a free port")
	}
	if rows := parseTable([]byte{5, 0, 0, 0, 1}); len(rows) != 0 {
		t.Fatalf("a short table gave rows %+v", rows)
	}
}

func TestLabel(t *testing.T) {
	if got := (Holder{PID: 4120, Program: "s7.exe"}).Label(); got != "s7.exe (process 4120)" {
		t.Fatal(got)
	}
	if got := (Holder{PID: 9}).Label(); got != "the program with process id 9" {
		t.Fatal(got)
	}
}
