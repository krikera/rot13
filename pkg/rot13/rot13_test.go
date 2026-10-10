package rot13

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestTransformByte(t *testing.T) {
	tests := []struct {
		name     string
		input    byte
		expected byte
	}{
		{"lowercase a -> n", 'a', 'n'},
		{"lowercase m -> z", 'm', 'z'},
		{"lowercase n -> a", 'n', 'a'},
		{"lowercase z -> m", 'z', 'm'},
		{"uppercase A -> N", 'A', 'N'},
		{"uppercase M -> Z", 'M', 'Z'},
		{"uppercase N -> A", 'N', 'A'},
		{"uppercase Z -> M", 'Z', 'M'},
		{"digit 0 unchanged", '0', '0'},
		{"digit 9 unchanged", '9', '9'},
		{"punctuation unchanged", '!', '!'},
		{"whitespace unchanged", ' ', ' '},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TransformByte(tt.input)
			if got != tt.expected {
				t.Errorf("TransformByte(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTransformStream(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard sentence",
			input:    "Hello, World!",
			expected: "Uryyb, Jbeyq!",
		},
		{
			name:     "full alphabet lowercase",
			input:    "abcdefghijklmnopqrstuvwxyz",
			expected: "nopqrstuvwxyzabcdefghijklm",
		},
		{
			name:     "full alphabet uppercase",
			input:    "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			expected: "NOPQRSTUVWXYZABCDEFGHIJKLM",
		},
		{
			name:     "multiline with symbols",
			input:    "123-ABC\nxyz: 456\n",
			expected: "123-NOP\nklm: 456\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var w bytes.Buffer
			if err := TransformStream(r, &w); err != nil {
				t.Fatalf("TransformStream error: %v", err)
			}
			if got := w.String(); got != tt.expected {
				t.Errorf("TransformStream(%q) = %q; want %q", tt.input, got, tt.expected)
			}

			// Invertibility invariant: rot13(rot13(x)) == x
			var roundTrip bytes.Buffer
			if err := TransformStream(strings.NewReader(w.String()), &roundTrip); err != nil {
				t.Fatalf("roundtrip error: %v", err)
			}
			if roundTrip.String() != tt.input {
				t.Errorf("Roundtrip failed: got %q, want %q", roundTrip.String(), tt.input)
			}
		})
	}
}

func TestUTF8MultibytePreservation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "mixed French and emoji",
			input:    "Café au lait ☕ with crêpes! 🥐",
			expected: "Pnsé nh ynvg ☕ jvgu peêcrf! 🥐",
		},
		{
			name:     "German umlauts and eszett",
			input:    "Größe und Übermut: Straße",
			expected: "Teößr haq Üorezhg: Fgenßr",
		},
		{
			name:     "CJK characters and English brand",
			input:    "Go语言 (Go Language) 🚀 2026",
			expected: "Tb语言 (Tb Ynathntr) 🚀 2026",
		},
		{
			name:     "Cyrillic script with English names",
			input:    "User John: Привет, как дела?",
			expected: "Hfre Wbua: Привет, как дела?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w bytes.Buffer
			if err := TransformStream(strings.NewReader(tt.input), &w); err != nil {
				t.Fatalf("TransformStream error: %v", err)
			}
			if got := w.String(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}

			var restored bytes.Buffer
			if err := TransformStream(strings.NewReader(w.String()), &restored); err != nil {
				t.Fatalf("roundtrip error: %v", err)
			}
			if restored.String() != tt.input {
				t.Errorf("restored = %q, want original %q", restored.String(), tt.input)
			}
		})
	}
}

func TestBinaryDataTransparency(t *testing.T) {
	raw := make([]byte, 256)
	for i := 0; i < 256; i++ {
		raw[i] = byte(i)
	}

	var encoded bytes.Buffer
	if err := TransformStream(bytes.NewReader(raw), &encoded); err != nil {
		t.Fatalf("failed to encode raw binary: %v", err)
	}

	encodedBytes := encoded.Bytes()
	for i := 0; i < 256; i++ {
		b := byte(i)
		isLetter := (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
		if !isLetter && encodedBytes[i] != b {
			t.Errorf("byte 0x%02X changed to 0x%02X", b, encodedBytes[i])
		}
	}

	var decoded bytes.Buffer
	if err := TransformStream(bytes.NewReader(encodedBytes), &decoded); err != nil {
		t.Fatalf("failed to decode binary stream: %v", err)
	}

	if !bytes.Equal(decoded.Bytes(), raw) {
		t.Errorf("binary round-trip failed to reproduce original payload")
	}
}

func TestLargeStreamBoundary(t *testing.T) {
	const size = 5 * 1024 * 1024
	payload := make([]byte, size)
	for i := 0; i < size; i++ {
		payload[i] = byte('A' + (i % 26))
	}

	var encoded bytes.Buffer
	if err := TransformStream(bytes.NewReader(payload), &encoded); err != nil {
		t.Fatalf("large stream encode failed: %v", err)
	}

	if encoded.Len() != size {
		t.Fatalf("encoded length %d does not match original size %d", encoded.Len(), size)
	}

	var decoded bytes.Buffer
	if err := TransformStream(bytes.NewReader(encoded.Bytes()), &decoded); err != nil {
		t.Fatalf("large stream decode failed: %v", err)
	}

	if !bytes.Equal(decoded.Bytes(), payload) {
		t.Fatalf("large stream round-trip mismatch across buffer boundaries")
	}
}

func TestEmptyStream(t *testing.T) {
	var out bytes.Buffer
	if err := TransformStream(bytes.NewReader([]byte{}), &out); err != nil {
		t.Fatalf("empty stream returned unexpected error: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected 0 bytes, got %d", out.Len())
	}
}

type faultyReader struct {
	err error
}

func (r *faultyReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

type faultyWriter struct {
	err error
}

func (w *faultyWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

func TestTransformStream_ReaderError(t *testing.T) {
	expectedErr := io.ErrUnexpectedEOF
	r := &faultyReader{err: expectedErr}
	var w bytes.Buffer

	err := TransformStream(r, &w)
	if err == nil {
		t.Fatalf("expected error from faulty reader, got nil")
	}
	if err != expectedErr {
		t.Errorf("got error %v, want %v", err, expectedErr)
	}
}

func TestTransformStream_WriterError(t *testing.T) {
	expectedErr := io.ErrShortWrite
	r := strings.NewReader("sample payload")
	w := &faultyWriter{err: expectedErr}

	err := TransformStream(r, w)
	if err == nil {
		t.Fatalf("expected error from faulty writer, got nil")
	}
	if err != expectedErr {
		t.Errorf("got error %v, want %v", err, expectedErr)
	}
}

func TestTransformStream_ExactBufferBoundaries(t *testing.T) {
	boundarySizes := []int{
		1,
		32*1024 - 1, // 32767
		32 * 1024,   // 32768
		32*1024 + 1, // 32769
		64 * 1024,   // 65536
		64*1024 + 1, // 65537
	}

	for _, sz := range boundarySizes {
		payload := make([]byte, sz)
		for i := 0; i < sz; i++ {
			payload[i] = byte('A' + (i % 26))
		}

		var enc bytes.Buffer
		if err := TransformStream(bytes.NewReader(payload), &enc); err != nil {
			t.Fatalf("size %d: failed to encode: %v", sz, err)
		}
		if enc.Len() != sz {
			t.Fatalf("size %d: encoded length %d != expected %d", sz, enc.Len(), sz)
		}

		var dec bytes.Buffer
		if err := TransformStream(bytes.NewReader(enc.Bytes()), &dec); err != nil {
			t.Fatalf("size %d: failed to decode: %v", sz, err)
		}
		if !bytes.Equal(dec.Bytes(), payload) {
			t.Fatalf("size %d: roundtrip mismatch across boundary", sz)
		}
	}
}

type chunkedReader struct {
	data      []byte
	chunkSize int
	pos       int
}

func (r *chunkedReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	remaining := len(r.data) - r.pos
	toRead := r.chunkSize
	if toRead > remaining {
		toRead = remaining
	}
	if toRead > len(p) {
		toRead = len(p)
	}
	copy(p, r.data[r.pos:r.pos+toRead])
	r.pos += toRead
	return toRead, nil
}

func TestTransformStream_ChunkedReads(t *testing.T) {
	const totalSize = 100 * 1024
	payload := make([]byte, totalSize)
	for i := 0; i < totalSize; i++ {
		payload[i] = byte('a' + (i % 26))
	}

	chunkSizes := []int{1, 7, 13, 1023, 4096}
	for _, cs := range chunkSizes {
		r := &chunkedReader{data: payload, chunkSize: cs}
		var enc bytes.Buffer
		if err := TransformStream(r, &enc); err != nil {
			t.Fatalf("chunk size %d: encode failed: %v", cs, err)
		}

		rDec := &chunkedReader{data: enc.Bytes(), chunkSize: cs}
		var dec bytes.Buffer
		if err := TransformStream(rDec, &dec); err != nil {
			t.Fatalf("chunk size %d: decode failed: %v", cs, err)
		}

		if !bytes.Equal(dec.Bytes(), payload) {
			t.Fatalf("chunk size %d: chunked stream roundtrip mismatch", cs)
		}
	}
}

func TestTransformByte_Exhaustive256(t *testing.T) {
	for i := 0; i < 256; i++ {
		b := byte(i)
		got := TransformByte(b)
		var want byte
		switch {
		case b >= 'a' && b <= 'm':
			want = b + 13
		case b >= 'n' && b <= 'z':
			want = b - 13
		case b >= 'A' && b <= 'M':
			want = b + 13
		case b >= 'N' && b <= 'Z':
			want = b - 13
		default:
			want = b
		}
		if got != want {
			t.Errorf("byte 0x%02X (%q): got 0x%02X, want 0x%02X", b, b, got, want)
		}

		// Double transform must restore original byte
		if TransformByte(got) != b {
			t.Errorf("byte 0x%02X: double transform did not invert", b)
		}
	}
}

func BenchmarkTransformByte(b *testing.B) {
	input := byte('m')
	for i := 0; i < b.N; i++ {
		_ = TransformByte(input)
	}
}

func BenchmarkTransformStream1MB(b *testing.B) {
	data := make([]byte, 1024*1024)
	for i := 0; i < len(data); i++ {
		data[i] = byte('A' + (i % 26))
	}
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		w := io.Discard
		_ = TransformStream(r, w)
	}
}
