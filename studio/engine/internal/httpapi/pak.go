package httpapi

import (
	"bytes"
	"compress/zlib"
	"crypto/sha3"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"

	"studio-engine/internal/pyjson"
)

// frontend.pak (launcher/frontend_assets.py, written by studio/tools/pack_frontend.py):
// "SSPAK1" + 16-byte nonce + body XOR SHAKE-256(key | nonce); body = <u32 LE header
// length><compact JSON {name: [offset, length]}><zlib blobs>. The key only deters casual
// copying; it is the same constant as in the Python module.
const pakMagic = "SSPAK1"

var pakKey, _ = hex.DecodeString("8c1e5a7f03d94b6e2fa1c0d7e95b3418a6f20c7d51e9b38f4a0d6c2e97b15f83")

// unpackFrontend returns the files of a frontend.pak.
func unpackFrontend(blob []byte) (map[string][]byte, error) {
	if !bytes.HasPrefix(blob, []byte(pakMagic)) || len(blob) < len(pakMagic)+16+4 {
		return nil, errors.New("frontend.pak is not a ScadaBench frontend package")
	}
	nonce := blob[len(pakMagic) : len(pakMagic)+16]
	body := append([]byte(nil), blob[len(pakMagic)+16:]...)
	shake := sha3.NewSHAKE256()
	shake.Write(pakKey)
	shake.Write(nonce)
	pad := make([]byte, len(body))
	_, _ = shake.Read(pad)
	for index := range body {
		body[index] ^= pad[index]
	}
	size := int(binary.LittleEndian.Uint32(body))
	if 4+size > len(body) {
		return nil, errors.New("frontend.pak header is damaged")
	}
	decoded, err := pyjson.Decode(body[4 : 4+size])
	if err != nil {
		return nil, err
	}
	index, isObject := decoded.(*pyjson.Object)
	if !isObject {
		return nil, errors.New("frontend.pak header is damaged")
	}
	data := body[4+size:]
	files := map[string][]byte{}
	for _, name := range index.Keys() {
		entry, _ := index.Get(name)
		span, _ := entry.([]any)
		if len(span) != 2 {
			return nil, errors.New("frontend.pak index is damaged")
		}
		start, okStart := span[0].(int64)
		length, okLength := span[1].(int64)
		if !okStart || !okLength || start < 0 || length < 0 || start+length > int64(len(data)) {
			return nil, errors.New("frontend.pak index is damaged")
		}
		reader, err := zlib.NewReader(bytes.NewReader(data[start : start+length]))
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			return nil, err
		}
		files[name] = content
	}
	return files, nil
}
