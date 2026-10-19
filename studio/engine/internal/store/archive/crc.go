package archive

// crcPoly is the reflected CRC-32 (IEEE) polynomial used by ZIP, zlib and hash/crc32.
const crcPoly = 0xedb88320

// x2nTable[k] is x^(2^k) modulo the CRC polynomial, as zlib builds it.
var x2nTable = func() (table [32]uint32) {
	p := uint32(1) << 30 // x^1
	table[0] = p
	for n := 1; n < 32; n++ {
		p = multModP(p, p)
		table[n] = p
	}
	return table
}()

// multModP multiplies a and b modulo the CRC polynomial (zlib crc32.c multmodp).
func multModP(a, b uint32) uint32 {
	m := uint32(1) << 31
	var p uint32
	for {
		if a&m != 0 {
			p ^= b
			if a&(m-1) == 0 {
				break
			}
		}
		m >>= 1
		if b&1 != 0 {
			b = b>>1 ^ crcPoly
		} else {
			b >>= 1
		}
	}
	return p
}

// x2nModP returns x^(n * 2^k) modulo the CRC polynomial (zlib crc32.c x2nmodp).
func x2nModP(n int64, k uint) uint32 {
	p := uint32(1) << 31 // x^0
	for n != 0 {
		if n&1 != 0 {
			p = multModP(x2nTable[k&31], p)
		}
		n >>= 1
		k++
	}
	return p
}

// CombineCRC returns the CRC-32 of A followed by B from crc(A), crc(B) and len(B), exactly
// as zlib's crc32_combine64: the CRC of the concatenation never needs the bytes again.
func CombineCRC(crcA, crcB uint32, lenB int64) uint32 {
	return multModP(x2nModP(lenB, 3), crcA) ^ crcB
}
