package licensing

import "errors"

// ErrBase64 stands for the binascii.Error (a ValueError) of a bad base64 text.
var ErrBase64 = errors.New("Incorrect padding")

const base64Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// URLSafeB64Decode is Python's base64.urlsafe_b64decode(text) for a str: non-ASCII text
// fails, '-' and '_' stand for '+' and '/', other characters outside the alphabet are
// skipped, and a complete pad sequence ends the input (binascii non-strict mode).
func URLSafeB64Decode(text string) ([]byte, error) {
	var table [256]byte
	for index := range table {
		table[index] = 0xff
	}
	for index := 0; index < len(base64Alphabet); index++ {
		table[base64Alphabet[index]] = byte(index)
	}
	table['-'], table['_'] = 62, 63
	out := make([]byte, 0, len(text)*3/4+3)
	quad, pads := 0, 0
	var left byte
	for index := 0; index < len(text); index++ {
		c := text[index]
		if c >= 0x80 {
			return nil, errors.New("string argument should contain only ASCII characters")
		}
		if c == '=' {
			if quad >= 2 {
				pads++
				if quad+pads >= 4 {
					return out, nil
				}
			}
			continue
		}
		value := table[c]
		if value == 0xff {
			continue
		}
		pads = 0
		switch quad {
		case 0:
			left = value
		case 1:
			out = append(out, left<<2|value>>4)
			left = value & 0x0f
		case 2:
			out = append(out, left<<4|value>>2)
			left = value & 0x03
		case 3:
			out = append(out, left<<6|value)
		}
		quad = (quad + 1) % 4
	}
	if quad != 0 {
		return nil, ErrBase64
	}
	return out, nil
}

// PadBase64 appends "=" * (-len(text) % 4), counting characters as Python does.
func PadBase64(text string) string {
	length := len([]rune(text))
	for length%4 != 0 {
		text += "="
		length++
	}
	return text
}
