package rot13

import (
	"io"
)

// lut is a precomputed 256-byte lookup table for branchless ROT13 transformation.
var lut = func() [256]byte {
	var table [256]byte
	for i := 0; i < 256; i++ {
		b := byte(i)
		switch {
		case b >= 'a' && b <= 'z':
			table[i] = 'a' + (b-'a'+13)%26
		case b >= 'A' && b <= 'Z':
			table[i] = 'A' + (b-'A'+13)%26
		default:
			table[i] = b
		}
	}
	return table
}()

// TransformByte applies ROT13 substitution to a single byte via direct table lookup.
func TransformByte(b byte) byte {
	return lut[b]
}

// TransformStream reads from r and writes ROT13-encoded data to w in 32KB buffer chunks.
func TransformStream(r io.Reader, w io.Writer) error {
	buf := make([]byte, 32*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				buf[i] = lut[buf[i]]
			}
			if _, wErr := w.Write(buf[:n]); wErr != nil {
				return wErr
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
