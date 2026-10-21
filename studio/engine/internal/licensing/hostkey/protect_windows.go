//go:build windows

package hostkey

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// dpapiEntropy ties the blob to ScadaBench: another program of the same user does not open it
// by accident (this is not a defence against code running as the same user).
var dpapiEntropy = []byte("scadabench-host-key-v1")

const cryptProtectUIForbidden = 0x1

// dpapi is CryptProtectData / CryptUnprotectData in the current user's scope.
type dpapi struct{}

// System is the real DPAPI of this Windows user.
func System() Protector { return dpapi{} }

func blobOf(data []byte) *windows.DataBlob {
	if len(data) == 0 {
		return &windows.DataBlob{}
	}
	return &windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
}

func takeBlob(blob *windows.DataBlob) []byte {
	if blob.Data == nil {
		return nil
	}
	data := append([]byte(nil), unsafe.Slice(blob.Data, blob.Size)...)
	windows.LocalFree(windows.Handle(unsafe.Pointer(blob.Data)))
	return data
}

func (dpapi) Protect(plain []byte) ([]byte, error) {
	var out windows.DataBlob
	description, _ := windows.UTF16PtrFromString("ScadaBench virtual machine key")
	err := windows.CryptProtectData(blobOf(plain), description, blobOf(dpapiEntropy), 0, nil,
		cryptProtectUIForbidden, &out)
	if err != nil {
		return nil, err
	}
	return takeBlob(&out), nil
}

func (dpapi) Unprotect(blob []byte) ([]byte, error) {
	var out windows.DataBlob
	err := windows.CryptUnprotectData(blobOf(blob), nil, blobOf(dpapiEntropy), 0, nil, cryptProtectUIForbidden, &out)
	if err != nil {
		return nil, ErrUnprotect
	}
	return takeBlob(&out), nil
}
