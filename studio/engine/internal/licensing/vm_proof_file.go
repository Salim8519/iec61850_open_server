package licensing

import (
	"encoding/json"
	"os"
	"path/filepath"

	"studio-engine/internal/licensing/hostkey"
)

// VMProofFile is vm_proof.json in the license home: what the native workers read to accept a
// virtual machine whose host confirmed it. The workers accept it only for their own machine ID and
// only while Time, which the host signed, is at most 24 hours old (native/common/vm_proof.cpp).
const VMProofFile = "vm_proof.json"

type vmProofDocument struct {
	Certificate json.RawMessage `json:"certificate"`
	Nonce       string          `json:"nonce"`
	VMHWID      string          `json:"vm_hwid"`
	Signature   string          `json:"signature"`
	Time        int64           `json:"time"`
}

// writeVMProof stores the host's answer for the native workers, replacing the old proof atomically.
// Time is the time the host signed, so the file cannot be made younger by editing it.
func (s *Service) writeVMProof(answer hostkey.Answer, request hostkey.Request) error {
	data, err := json.MarshalIndent(vmProofDocument{Certificate: answer.Certificate, Nonce: request.Nonce,
		VMHWID: request.VMHWID, Signature: answer.Signature, Time: answer.Time}, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(s.options.Home, VMProofFile), data)
}

// writeFileAtomic replaces path through a temporary file so a reader never sees half of it.
func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o666); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
