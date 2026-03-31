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
	reverseSigil := &ReverseSigil{}

	out, err := reverseSigil.In([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, []byte("olleh"), out)

	restored, err := reverseSigil.Out(out)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), restored)
}

func TestSigil_ReverseSigil_Bad(t *testing.T) {
	reverseSigil := &ReverseSigil{}

	out, err := reverseSigil.In([]byte{})
	require.NoError(t, err)
	assert.Equal(t, []byte{}, out)
}

func TestSigil_ReverseSigil_NilInput_Good(t *testing.T) {
	reverseSigil := &ReverseSigil{}

	out, err := reverseSigil.In(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	out, err = reverseSigil.Out(nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestSigil_HexSigil_Good(t *testing.T) {
	hexSigil := &HexSigil{}
	data := []byte("hello world")

	encoded, err := hexSigil.In(data)
	require.NoError(t, err)
	assert.Equal(t, []byte(hex.EncodeToString(data)), encoded)

	decoded, err := hexSigil.Out(encoded)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestSigil_HexSigil_Bad(t *testing.T) {
	hexSigil := &HexSigil{}

	_, err := hexSigil.Out([]byte("zzzz"))
	assert.Error(t, err)

	out, err := hexSigil.In([]byte{})
	require.NoError(t, err)
	assert.Equal(t, []byte{}, out)
}

func TestSigil_HexSigil_NilInput_Good(t *testing.T) {
	hexSigil := &HexSigil{}

	out, err := hexSigil.In(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	out, err = hexSigil.Out(nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestSigil_Base64Sigil_Good(t *testing.T) {
	base64Sigil := &Base64Sigil{}
	data := []byte("composable transforms")

	encoded, err := base64Sigil.In(data)
	require.NoError(t, err)
	assert.Equal(t, []byte(base64.StdEncoding.EncodeToString(data)), encoded)

	decoded, err := base64Sigil.Out(encoded)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestSigil_Base64Sigil_Bad(t *testing.T) {
	base64Sigil := &Base64Sigil{}

	_, err := base64Sigil.Out([]byte("!!!"))
	assert.Error(t, err)

	out, err := base64Sigil.In([]byte{})
	require.NoError(t, err)
	assert.Equal(t, []byte{}, out)
}

func TestSigil_Base64Sigil_NilInput_Good(t *testing.T) {
	base64Sigil := &Base64Sigil{}

	out, err := base64Sigil.In(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	out, err = base64Sigil.Out(nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestSigil_GzipSigil_Good(t *testing.T) {
	gzipSigil := &GzipSigil{}
	data := []byte("the quick brown fox jumps over the lazy dog")

	compressed, err := gzipSigil.In(data)
	require.NoError(t, err)
	assert.NotEqual(t, data, compressed)

	decompressed, err := gzipSigil.Out(compressed)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed)
}

func TestSigil_GzipSigil_Bad(t *testing.T) {
	gzipSigil := &GzipSigil{}

	_, err := gzipSigil.Out([]byte("not gzip"))
	assert.Error(t, err)

	compressed, err := gzipSigil.In([]byte{})
	require.NoError(t, err)
	assert.NotEmpty(t, compressed)

	decompressed, err := gzipSigil.Out(compressed)
	require.NoError(t, err)
	assert.Equal(t, []byte{}, decompressed)
}

func TestSigil_GzipSigil_NilInput_Good(t *testing.T) {
	gzipSigil := &GzipSigil{}

	out, err := gzipSigil.In(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	out, err = gzipSigil.Out(nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestSigil_JSONSigil_Good(t *testing.T) {
	jsonSigil := &JSONSigil{Indent: false}
	data := []byte(`{  "key" :   "value"  }`)

	compacted, err := jsonSigil.In(data)
	require.NoError(t, err)
	assert.Equal(t, []byte(`{"key":"value"}`), compacted)

	passthrough, err := jsonSigil.Out(compacted)
	require.NoError(t, err)
	assert.Equal(t, compacted, passthrough)
}

func TestSigil_JSONSigil_Indent_Good(t *testing.T) {
	jsonSigil := &JSONSigil{Indent: true}
	data := []byte(`{"key":"value"}`)

	indented, err := jsonSigil.In(data)
	require.NoError(t, err)
	assert.Contains(t, string(indented), "\n")
	assert.Contains(t, string(indented), "  ")
}

func TestSigil_JSONSigil_Bad(t *testing.T) {
	jsonSigil := &JSONSigil{Indent: false}

	_, err := jsonSigil.In([]byte("not json"))
	assert.Error(t, err)
}

func TestSigil_JSONSigil_NilInput_Good(t *testing.T) {
	jsonSigil := &JSONSigil{Indent: false}

	out, err := jsonSigil.In(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	out, err = jsonSigil.Out(nil)
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
			sigilValue, err := NewSigil(tt.sigilName)
			require.NoError(t, err)

			hashed, err := sigilValue.In(data)
			require.NoError(t, err)
			assert.Len(t, hashed, tt.size)

			passthrough, err := sigilValue.Out(hashed)
			require.NoError(t, err)
			assert.Equal(t, hashed, passthrough)
		})
	}
}

func TestSigil_HashSigil_Bad(t *testing.T) {
	hashSigil := &HashSigil{Hash: 0}
	_, err := hashSigil.In([]byte("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

func TestSigil_HashSigil_EmptyInput_Good(t *testing.T) {
	sigilValue, err := NewSigil("sha256")
	require.NoError(t, err)

	hashed, err := sigilValue.In([]byte{})
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
			sigilValue, err := NewSigil(name)
			require.NoError(t, err)
			assert.NotNil(t, sigilValue)
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
