package signal

import "math/big"

// pyRandom is CPython's random.Random: MT19937 seeded with init_by_array over the 32-bit
// words of abs(seed), and random() built from two outputs (53-bit resolution).
type pyRandom struct {
	state [624]uint32
	index int
}

func newPyRandom(seed *big.Int) *pyRandom {
	magnitude := new(big.Int).Abs(seed)
	var key []uint32
	words := magnitude.Bits()
	for _, word := range words {
		key = append(key, uint32(word))
		if bitsPerWord == 64 {
			key = append(key, uint32(uint64(word)>>32))
		}
	}
	for len(key) > 1 && key[len(key)-1] == 0 {
		key = key[:len(key)-1]
	}
	if len(key) == 0 {
		key = []uint32{0}
	}
	r := &pyRandom{}
	r.initByArray(key)
	return r
}

const bitsPerWord = 32 << (^uint(0) >> 63)

func (r *pyRandom) initGenrand(seed uint32) {
	r.state[0] = seed
	for i := 1; i < 624; i++ {
		previous := r.state[i-1]
		r.state[i] = 1812433253*(previous^(previous>>30)) + uint32(i)
	}
	r.index = 624
}

func (r *pyRandom) initByArray(key []uint32) {
	r.initGenrand(19650218)
	i, j := 1, 0
	k := 624
	if len(key) > k {
		k = len(key)
	}
	for ; k > 0; k-- {
		previous := r.state[i-1]
		r.state[i] = (r.state[i] ^ ((previous ^ (previous >> 30)) * 1664525)) + key[j] + uint32(j)
		i++
		j++
		if i >= 624 {
			r.state[0] = r.state[623]
			i = 1
		}
		if j >= len(key) {
			j = 0
		}
	}
	for k = 623; k > 0; k-- {
		previous := r.state[i-1]
		r.state[i] = (r.state[i] ^ ((previous ^ (previous >> 30)) * 1566083941)) - uint32(i)
		i++
		if i >= 624 {
			r.state[0] = r.state[623]
			i = 1
		}
	}
	r.state[0] = 0x80000000
}

func (r *pyRandom) genrand() uint32 {
	if r.index >= 624 {
		for k := 0; k < 624; k++ {
			y := r.state[k]&0x80000000 | r.state[(k+1)%624]&0x7fffffff
			next := r.state[(k+397)%624] ^ (y >> 1)
			if y&1 != 0 {
				next ^= 0x9908b0df
			}
			r.state[k] = next
		}
		r.index = 0
	}
	y := r.state[r.index]
	r.index++
	y ^= y >> 11
	y ^= (y << 7) & 0x9d2c5680
	y ^= (y << 15) & 0xefc60000
	y ^= y >> 18
	return y
}

// random is random.random().
func (r *pyRandom) random() float64 {
	a, b := r.genrand()>>5, r.genrand()>>6
	// float64(...) around each product forbids fused multiply-add, as CPython rounds each step.
	return (float64(float64(a)*67108864.0) + float64(b)) * (1.0 / 9007199254740992.0)
}

// uniform is random.uniform(a, b) = a + (b - a) * random().
func (r *pyRandom) uniform(a, b float64) float64 { return a + float64((b-a)*r.random()) }
