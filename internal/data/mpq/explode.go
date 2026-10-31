package mpq

import "errors"

// PKWARE Data Compression Library "implode" decoder. Port of the algorithm in
// zlib's contrib/blast (Mark Adler), which is the same format MPQ uses.

var (
	litLenRep  = []byte{11, 124, 8, 7, 28, 7, 188, 13, 76, 4, 10, 8, 12, 10, 12, 10, 8, 23, 8, 9, 7, 6, 7, 8, 7, 6, 55, 8, 23, 24, 12, 11, 7, 9, 11, 12, 6, 7, 22, 5, 7, 24, 6, 11, 9, 6, 7, 22, 7, 11, 38, 7, 9, 8, 25, 11, 8, 11, 9, 12, 8, 12, 5, 38, 5, 38, 5, 11, 7, 5, 6, 21, 6, 10, 53, 8, 7, 24, 10, 27, 44, 253, 253, 253, 252, 252, 252, 13, 12, 45, 12, 45, 12, 61, 12, 45, 44, 173}
	lenLenRep  = []byte{2, 35, 36, 53, 38, 23}
	distLenRep = []byte{2, 20, 53, 230, 247, 151, 248}
	lenBase    = [16]int{3, 2, 4, 5, 6, 7, 8, 9, 10, 12, 16, 24, 40, 72, 136, 264}
	lenExtra   = [16]uint{0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8}

	litCode, lenCode, distCode = newHuff(litLenRep), newHuff(lenLenRep), newHuff(distLenRep)
)

const maxBits = 13

type huff struct {
	count  [maxBits + 1]int
	symbol []int
}

func newHuff(rep []byte) *huff {
	var lengths []int
	for _, b := range rep {
		n := int(b>>4) + 1
		for ; n > 0; n-- {
			lengths = append(lengths, int(b&15))
		}
	}
	h := &huff{}
	for _, l := range lengths {
		h.count[l]++
	}
	var offs [maxBits + 2]int
	for l := 1; l < maxBits; l++ {
		offs[l+1] = offs[l] + h.count[l]
	}
	h.symbol = make([]int, len(lengths))
	for s, l := range lengths {
		if l != 0 {
			h.symbol[offs[l]] = s
			offs[l]++
		}
	}
	return h
}

type bitReader struct {
	in  []byte
	pos int
	buf uint32
	cnt uint
}

var errTruncated = errors.New("mpq: truncated implode stream")

func (b *bitReader) bits(n uint) (int, error) {
	for b.cnt < n {
		if b.pos >= len(b.in) {
			return 0, errTruncated
		}
		b.buf |= uint32(b.in[b.pos]) << b.cnt
		b.pos++
		b.cnt += 8
	}
	v := int(b.buf & (1<<n - 1))
	b.buf >>= n
	b.cnt -= n
	return v, nil
}

func (b *bitReader) decode(h *huff) (int, error) {
	code, first, index := 0, 0, 0
	for l := 1; l <= maxBits; l++ {
		bit, err := b.bits(1)
		if err != nil {
			return 0, err
		}
		code |= bit ^ 1
		count := h.count[l]
		if code < first+count {
			return h.symbol[index+code-first], nil
		}
		index += count
		first = (first + count) << 1
		code <<= 1
	}
	return 0, errors.New("mpq: bad implode code")
}

// explode decompresses a PKWARE DCL stream. size is the expected output size
// (used as capacity hint; the end code terminates the stream).
func explode(in []byte, size int) ([]byte, error) {
	br := &bitReader{in: in}
	lit, err := br.bits(8)
	if err != nil {
		return nil, err
	}
	dict, err := br.bits(8)
	if err != nil {
		return nil, err
	}
	if lit > 1 || dict < 4 || dict > 6 {
		return nil, errors.New("mpq: bad implode header")
	}
	out := make([]byte, 0, size)
	for {
		flag, err := br.bits(1)
		if err != nil {
			return out, err
		}
		if flag == 0 {
			var c int
			if lit == 1 {
				c, err = br.decode(litCode)
			} else {
				c, err = br.bits(8)
			}
			if err != nil {
				return out, err
			}
			out = append(out, byte(c))
			continue
		}
		sym, err := br.decode(lenCode)
		if err != nil {
			return out, err
		}
		x, err := br.bits(lenExtra[sym])
		if err != nil {
			return out, err
		}
		n := lenBase[sym] + x
		if n == 519 {
			return out, nil
		}
		shift := uint(dict)
		if n == 2 {
			shift = 2
		}
		d, err := br.decode(distCode)
		if err != nil {
			return out, err
		}
		low, err := br.bits(shift)
		if err != nil {
			return out, err
		}
		dist := d<<shift + low + 1
		if dist > len(out) {
			return out, errors.New("mpq: implode distance too far back")
		}
		for ; n > 0; n-- {
			out = append(out, out[len(out)-dist])
		}
	}
}
