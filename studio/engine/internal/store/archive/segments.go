package archive

import (
	"bytes"
	"compress/flate"
	"hash/crc32"
	"runtime"
	"sync"
)

// chunkSize splits one large SCL text into independently compressed pieces so a cold save
// uses every core. Each piece ends with a sync flush, so the pieces concatenate.
const chunkSize = 1 << 20

// segment is a run of project.json bytes compressed on its own: no back-reference crosses
// its borders and its deflate data ends byte-aligned with a non-final sync-flush block.
type segment struct {
	raw        []byte // uncompressed bytes; dropped after compression for SCL segments
	keepRaw    bool   // literal segments keep raw bytes to recognise an unchanged value
	crc        uint32 // CRC-32 of raw
	size       int64  // len(raw)
	once       sync.Once
	compressed []byte
}

func newSegment(raw []byte, keepRaw bool) *segment {
	return &segment{raw: raw, keepRaw: keepRaw, crc: crc32.ChecksumIEEE(raw), size: int64(len(raw))}
}

var writers = sync.Pool{New: func() any {
	w, _ := flate.NewWriter(nil, flate.DefaultCompression) // zlib's default level 6
	return w
}}

var compressSlots = make(chan struct{}, max(1, runtime.GOMAXPROCS(0)))

// deflateFlushed compresses raw and ends with a sync flush (never a final block).
func deflateFlushed(raw []byte) []byte {
	compressSlots <- struct{}{}
	defer func() { <-compressSlots }()
	var out bytes.Buffer
	out.Grow(len(raw)/4 + 64)
	w := writers.Get().(*flate.Writer)
	w.Reset(&out)
	_, _ = w.Write(raw) // writing to a bytes.Buffer cannot fail
	_ = w.Flush()
	writers.Put(w)
	return out.Bytes()
}

// compress fills s.compressed once, in parallel chunks for large segments.
func (s *segment) compress() {
	s.once.Do(func() {
		if len(s.raw) <= chunkSize {
			s.compressed = deflateFlushed(s.raw)
		} else {
			parts := make([][]byte, (len(s.raw)+chunkSize-1)/chunkSize)
			var wait sync.WaitGroup
			for index := range parts {
				start := index * chunkSize
				end := min(start+chunkSize, len(s.raw))
				wait.Add(1)
				go func() {
					defer wait.Done()
					parts[index] = deflateFlushed(s.raw[start:end])
				}()
			}
			wait.Wait()
			s.compressed = bytes.Join(parts, nil)
		}
		if !s.keepRaw {
			s.raw = nil
		}
	})
}

// compressAll compresses every segment that is not compressed yet, concurrently.
func compressAll(segments []*segment) [][]byte {
	var wait sync.WaitGroup
	for _, s := range segments {
		wait.Add(1)
		go func() {
			defer wait.Done()
			s.compress()
		}()
	}
	wait.Wait()
	out := make([][]byte, len(segments))
	for index, s := range segments {
		out[index] = s.compressed
	}
	return out
}

// combine returns the CRC-32 and length of the concatenated segments.
func combine(segments []*segment) (uint32, int64) {
	var crc uint32
	var size int64
	for _, s := range segments {
		crc = CombineCRC(crc, s.crc, s.size)
		size += s.size
	}
	return crc, size
}
