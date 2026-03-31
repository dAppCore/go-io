package sigil

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSigil_ReverseSigil_Good(t *testing.T) {
	s := &ReverseSigil{}

	out, err := s.In([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, []byte("olleh"), out)

	restored, err := s.Out(out)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), restored)
}

func TestSigil_ReverseSigil_Bad(t *testing.T) {
	s := &ReverseSigil{}

	out, err := s.In([]byte{})
	require.NoError(t, err)
	assert.Equal(t, []byte{}, out)
}

func TestSigil_ReverseSigil_NilInput_Good(t *testing.T) {
	s := &ReverseSigil{}

	out, err := s.In(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	out, err = s.Out(nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestSigil_HexSigil_Good(t *testing.T) {
	s := &HexSigil{}
	data := []byte("hello world")

	encoded, err := s.In(data)
	require.NoError(t, err)
	assert.Equal(t, []byte(hex.EncodeToString(data)), encoded)

	decoded, err := s.Out(encoded)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestSigil_HexSigil_Bad(t *testing.T) {
	s := &HexSigil{}

	_, err := s.Out([]byte("zzzz"))
	assert.Error(t, err)

	out, err := s.In([]byte{})
	require.NoError(t, err)
	assert.Equal(t, []byte{}, out)
}

func TestSigil_HexSigil_NilInput_Good(t *testing.T) {
	s := &HexSigil{}

	out, err := s.In(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	out, err = s.Out(nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestSigil_Base64Sigil_Good(t *testing.T) {
	s := &Base64Sigil{}
	data := []byte("composable transforms")

	encoded, err := s.In(data)
	require.NoError(t, err)
	assert.Equal(t, []byte(base64.StdEncoding.EncodeToString(data)), encoded)

	decoded, err := s.Out(encoded)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestSigil_Base64Sigil_Bad(t *testing.T) {
	s := &Base64Sigil{}

	_, err := s.Out([]byte("!!!"))
	assert.Error(t, err)

	out, err := s.In([]byte{})
	require.NoError(t, err)
	assert.Equal(t, []byte{}, out)
}

func TestSigil_Base64Sigil_NilInput_Good(t *testing.T) {
	s := &Base64Sigil{}

	out, err := s.In(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	out, err = s.Out(nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestSigil_GzipSigil_Good(t *testing.T) {
	s := &GzipSigil{}
	data := []byte("the quick brown fox jumps over the lazy dog")

	compressed, err := s.In(data)
	require.NoError(t, err)
	assert.NotEqual(t, data, compressed)

	decompressed, err := s.Out(compressed)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed)
}

func TestSigil_GzipSigil_Bad(t *testing.T) {
	s := &GzipSigil{}

	_, err := s.Out([]byte("not gzip"))
	assert.Error(t, err)

	compressed, err := s.In([]byte{})
	require.NoError(t, err)
	assert.NotEmpty(t, compressed)

	decompressed, err := s.Out(compressed)
	require.NoError(t, err)
	assert.Equal(t, []byte{}, decompressed)
}

func TestSigil_GzipSigil_NilInput_Good(t *testing.T) {
	s := &GzipSigil{}

	out, err := s.In(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	out, err = s.Out(nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestSigil_JSONSigil_Good(t *testing.T) {
	s := &JSONSigil{Indent: false}
	data := []byte(`{  "key" :   "value"  }`)

	compacted, err := s.In(data)
	require.NoError(t, err)
	assert.Equal(t, []byte(`{"key":"value"}`), compacted)

	passthrough, err := s.Out(compacted)
	require.NoError(t, err)
	assert.Equal(t, compacted, passthrough)
}

func TestSigil_JSONSigil_Indent_Good(t *testing.T) {
	s := &JSONSigil{Indent: true}
	data := []byte(`{"key":"value"}`)

	indented, err := s.In(data)
	require.NoError(t, err)
	assert.Contains(t, string(indented), "\n")
	assert.Contains(t, string(indented), "  ")
}

func TestSigil_JSONSigil_Bad(t *testing.T) {
	s := &JSONSigil{Indent: false}

	_, err := s.In([]byte("not json"))
	assert.Error(t, err)
}

func TestSigil_JSONSigil_NilInput_Good(t *testing.T) {
	s := &JSONSigil{Indent: false}

	out, err := s.In(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	out, err = s.Out(nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestSigil_HashSigil_Good(t *testing.T) {
	data := []byte("hash me")

	tests := []struct {
		name      string
		sigilName string
		size      int
	}{
		{"md5", "md5", md5.Size},
		{"sha1", "sha1", sha1.Size},
		{"sha256", "sha256", sha256.Size},
		{"sha512", "sha512", sha512.Size},
		{"sha224", "sha224", sha256.Size224},
		{"sha384", "sha384", sha512.Size384},
		{"sha512-224", "sha512-224", 28},
		{"sha512-256", "sha512-256", 32},
		{"sha3-224", "sha3-224", 28},
		{"sha3-256", "sha3-256", 32},
		{"sha3-384", "sha3-384", 48},
		{"sha3-512", "sha3-512", 64},
		{"ripemd160", "ripemd160", 20},
		{"blake2s-256", "blake2s-256", 32},
		{"blake2b-256", "blake2b-256", 32},
		{"blake2b-384", "blake2b-384", 48},
		{"blake2b-512", "blake2b-512", 64},
		{"md4", "md4", 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewSigil(tt.sigilName)
			require.NoError(t, err)

			hashed, err := s.In(data)
			require.NoError(t, err)
			assert.Len(t, hashed, tt.size)

			passthrough, err := s.Out(hashed)
			require.NoError(t, err)
			assert.Equal(t, hashed, passthrough)
		})
	}
}

func TestSigil_HashSigil_Bad(t *testing.T) {
	s := &HashSigil{Hash: 0}
	_, err := s.In([]byte("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestSigil_HashSigil_EmptyInput_Good(t *testing.T) {
	s, err := NewSigil("sha256")
	require.NoError(t, err)

	hashed, err := s.In([]byte{})
	require.NoError(t, err)
	assert.Len(t, hashed, sha256.Size)
}

func TestSigil_NewSigil_Good(t *testing.T) {
	names := []string{
		"reverse", "hex", "base64", "gzip", "json", "json-indent",
		"md4", "md5", "sha1", "sha224", "sha256", "sha384", "sha512",
		"ripemd160",
		"sha3-224", "sha3-256", "sha3-384", "sha3-512",
		"sha512-224", "sha512-256",
		"blake2s-256", "blake2b-256", "blake2b-384", "blake2b-512",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			s, err := NewSigil(name)
			require.NoError(t, err)
			assert.NotNil(t, s)
		})
	}
}

func TestSigil_NewSigil_Bad(t *testing.T) {
	_, err := NewSigil("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown sigil name")
}

func TestSigil_NewSigil_EmptyName_Bad(t *testing.T) {
	_, err := NewSigil("")
	assert.Error(t, err)
}

func TestSigil_Transmute_Good(t *testing.T) {
	data := []byte("round trip")

	hexSigil, err := NewSigil("hex")
	require.NoError(t, err)
	base64Sigil, err := NewSigil("base64")
	require.NoError(t, err)

	chain := []Sigil{hexSigil, base64Sigil}

	encoded, err := Transmute(data, chain)
	require.NoError(t, err)
	assert.NotEqual(t, data, encoded)

	decoded, err := Untransmute(encoded, chain)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestSigil_Transmute_MultiSigil_Good(t *testing.T) {
	data := []byte("multi sigil pipeline test data")

	reverseSigil, err := NewSigil("reverse")
	require.NoError(t, err)
	hexSigil, err := NewSigil("hex")
	require.NoError(t, err)
	base64Sigil, err := NewSigil("base64")
	require.NoError(t, err)

	chain := []Sigil{reverseSigil, hexSigil, base64Sigil}

	encoded, err := Transmute(data, chain)
	require.NoError(t, err)

	decoded, err := Untransmute(encoded, chain)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestSigil_Transmute_GzipRoundTrip_Good(t *testing.T) {
	data := []byte("compress then encode then decode then decompress")

	gzipSigil, err := NewSigil("gzip")
	require.NoError(t, err)
	hexSigil, err := NewSigil("hex")
	require.NoError(t, err)

	chain := []Sigil{gzipSigil, hexSigil}

	encoded, err := Transmute(data, chain)
	require.NoError(t, err)

	decoded, err := Untransmute(encoded, chain)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestSigil_Transmute_Bad(t *testing.T) {
	hexSigil := &HexSigil{}

	_, err := Untransmute([]byte("not-hex!!"), []Sigil{hexSigil})
	assert.Error(t, err)
}

func TestSigil_Transmute_NilAndEmptyInput_Good(t *testing.T) {
	data := []byte("unchanged")

	result, err := Transmute(data, nil)
	require.NoError(t, err)
	assert.Equal(t, data, result)

	result, err = Untransmute(data, nil)
	require.NoError(t, err)
	assert.Equal(t, data, result)

	hexSigil, _ := NewSigil("hex")
	result, err = Transmute(nil, []Sigil{hexSigil})
	require.NoError(t, err)
	assert.Nil(t, result)
}
