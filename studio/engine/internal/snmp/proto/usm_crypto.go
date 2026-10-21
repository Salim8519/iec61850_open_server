package proto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/hmac"
	"encoding/binary"
	"errors"
)

// USM processing failures; the agent answers each with its usmStats report.
var (
	ErrWrongDigest = errors.New("wrong digest")
	ErrDecryption  = errors.New("decryption error")
)

// mac is the truncated HMAC of a whole message whose digest field holds zeros.
func mac(keys *Keys, message []byte) []byte {
	spec := authSpecs[keys.Auth]
	h := hmac.New(spec.hash, keys.AuthKey)
	h.Write(message)
	return h.Sum(nil)[:spec.macLen]
}

// verify checks the digest of a received message.
func verify(keys *Keys, h *Header) error {
	spec, known := authSpecs[keys.Auth]
	if !known || len(h.AuthParams) != spec.macLen {
		return ErrWrongDigest
	}
	at := offsetIn(h.raw, h.AuthParams)
	if at < 0 || at+spec.macLen > len(h.raw) {
		return ErrWrongDigest
	}
	copyOf := append([]byte{}, h.raw...)
	clear(copyOf[at : at+spec.macLen])
	if !hmac.Equal(mac(keys, copyOf), h.AuthParams) {
		return ErrWrongDigest
	}
	return nil
}

// encrypt returns the privacy parameters and the encrypted scoped PDU.
func encrypt(keys *Keys, boots, clock int32, plain []byte, salt uint64) ([]byte, []byte, error) {
	if keys.Priv == "des" {
		params := make([]byte, 8)
		binary.BigEndian.PutUint32(params, uint32(boots))
		binary.BigEndian.PutUint32(params[4:], uint32(salt))
		block, err := des.NewCipher(keys.PrivKey[:8])
		if err != nil {
			return nil, nil, err
		}
		padded := append(append([]byte{}, plain...), make([]byte, (8-len(plain)%8)%8)...)
		out := make([]byte, len(padded))
		cipher.NewCBCEncrypter(block, desIV(keys, params)).CryptBlocks(out, padded)
		return params, out, nil
	}
	params := binary.BigEndian.AppendUint64(nil, salt)
	block, err := aes.NewCipher(keys.PrivKey)
	if err != nil {
		return nil, nil, err
	}
	out := make([]byte, len(plain))
	cfb(block, aesIV(boots, clock, params), out, plain, true)
	return params, out, nil
}

// decrypt returns the scoped PDU of a received message.
func decrypt(keys *Keys, h *Header) ([]byte, error) {
	if len(h.PrivParams) != 8 {
		return nil, ErrDecryption
	}
	if keys.Priv == "des" {
		if len(h.Encrypted)%8 != 0 || len(h.Encrypted) == 0 {
			return nil, ErrDecryption
		}
		block, err := des.NewCipher(keys.PrivKey[:8])
		if err != nil {
			return nil, ErrDecryption
		}
		out := make([]byte, len(h.Encrypted))
		cipher.NewCBCDecrypter(block, desIV(keys, h.PrivParams)).CryptBlocks(out, h.Encrypted)
		return out, nil
	}
	block, err := aes.NewCipher(keys.PrivKey)
	if err != nil {
		return nil, ErrDecryption
	}
	out := make([]byte, len(h.Encrypted))
	cfb(block, aesIV(h.Boots, h.Time, h.PrivParams), out, h.Encrypted, false)
	return out, nil
}

// desIV is the pre-IV (key bytes 8-15) XOR the salt (RFC 3414 8.1.1.1).
func desIV(keys *Keys, salt []byte) []byte {
	iv := make([]byte, 8)
	for index := range iv {
		iv[index] = keys.PrivKey[8+index] ^ salt[index]
	}
	return iv
}

// aesIV is boots, time and the 64-bit salt (RFC 3826 3.1.2.1).
func aesIV(boots, clock int32, salt []byte) []byte {
	iv := binary.BigEndian.AppendUint32(nil, uint32(boots))
	iv = binary.BigEndian.AppendUint32(iv, uint32(clock))
	return append(iv, salt...)
}

// cfb is 128-bit cipher feedback (RFC 3826 uses CFB128 without padding).
func cfb(block cipher.Block, iv, dst, src []byte, encrypting bool) {
	feedback := append([]byte{}, iv...)
	stream := make([]byte, aes.BlockSize)
	for start := 0; start < len(src); start += aes.BlockSize {
		block.Encrypt(stream, feedback)
		end := min(start+aes.BlockSize, len(src))
		for index := start; index < end; index++ {
			dst[index] = src[index] ^ stream[index-start]
		}
		if encrypting {
			copy(feedback, dst[start:end])
		} else {
			copy(feedback, src[start:end])
		}
	}
}
