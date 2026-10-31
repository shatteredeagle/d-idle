package d2s

// bitReader reads little-endian bit fields (least significant bit first), as the d2s stat block does.
type bitReader struct {
	b   []byte
	pos uint // bits consumed
}

func (r *bitReader) read(n uint) (uint32, bool) {
	if r.pos+n > uint(len(r.b))*8 {
		return 0, false
	}
	var v uint32
	for i := uint(0); i < n; i++ {
		bit := (r.b[(r.pos+i)/8] >> ((r.pos + i) % 8)) & 1
		v |= uint32(bit) << i
	}
	r.pos += n
	return v, true
}

func (r *bitReader) bytesUsed() int { return int((r.pos + 7) / 8) }

type bitWriter struct {
	b   []byte
	pos uint
}

func (w *bitWriter) write(v uint32, n uint) {
	for i := uint(0); i < n; i++ {
		if w.pos%8 == 0 {
			w.b = append(w.b, 0)
		}
		if v>>i&1 == 1 {
			w.b[len(w.b)-1] |= 1 << (w.pos % 8)
		}
		w.pos++
	}
}

func (w *bitWriter) bytes() []byte { return w.b }
