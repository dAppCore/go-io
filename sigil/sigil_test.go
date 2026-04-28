package sigil

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	. "dappco.re/go"
	"encoding/base64"
	"encoding/hex"
)

func TestSigil_ReverseSigil_Good(t *T) {
	reverseSigil := &ReverseSigil{}

	out, err := reverseSigil.In([]byte("hello"))
	RequireNoError(t, err)
	AssertEqual(t, []byte("olleh"), out)

	restored, err := reverseSigil.Out(out)
	RequireNoError(t, err)
	AssertEqual(t, []byte("hello"), restored)
}

func TestSigil_ReverseSigil_Bad(t *T) {
	reverseSigil := &ReverseSigil{}

	out, err := reverseSigil.In([]byte{})
	RequireNoError(t, err)
	AssertEqual(t, []byte{}, out)
}

func TestSigil_ReverseSigil_NilInput_Good(t *T) {
	reverseSigil := &ReverseSigil{}

	out, err := reverseSigil.In(nil)
	RequireNoError(t, err)
	AssertNil(t, out)

	out, err = reverseSigil.Out(nil)
	RequireNoError(t, err)
	AssertNil(t, out)
}

func TestSigil_HexSigil_Good(t *T) {
	hexSigil := &HexSigil{}
	data := []byte("hello world")

	encoded, err := hexSigil.In(data)
	RequireNoError(t, err)
	AssertEqual(t, []byte(hex.EncodeToString(data)), encoded)

	decoded, err := hexSigil.Out(encoded)
	RequireNoError(t, err)
	AssertEqual(t, data, decoded)
}

func TestSigil_HexSigil_Bad(t *T) {
	hexSigil := &HexSigil{}

	_, err := hexSigil.Out([]byte("zzzz"))
	AssertError(t, err)

	out, err := hexSigil.In([]byte{})
	RequireNoError(t, err)
	AssertEqual(t, []byte{}, out)
}

func TestSigil_HexSigil_NilInput_Good(t *T) {
	hexSigil := &HexSigil{}

	out, err := hexSigil.In(nil)
	RequireNoError(t, err)
	AssertNil(t, out)

	out, err = hexSigil.Out(nil)
	RequireNoError(t, err)
	AssertNil(t, out)
}

func TestSigil_Base64Sigil_Good(t *T) {
	base64Sigil := &Base64Sigil{}
	data := []byte("composable transforms")

	encoded, err := base64Sigil.In(data)
	RequireNoError(t, err)
	AssertEqual(t, []byte(base64.StdEncoding.EncodeToString(data)), encoded)

	decoded, err := base64Sigil.Out(encoded)
	RequireNoError(t, err)
	AssertEqual(t, data, decoded)
}

func TestSigil_Base64Sigil_Bad(t *T) {
	base64Sigil := &Base64Sigil{}

	_, err := base64Sigil.Out([]byte("!!!"))
	AssertError(t, err)

	out, err := base64Sigil.In([]byte{})
	RequireNoError(t, err)
	AssertEqual(t, []byte{}, out)
}

func TestSigil_Base64Sigil_NilInput_Good(t *T) {
	base64Sigil := &Base64Sigil{}

	out, err := base64Sigil.In(nil)
	RequireNoError(t, err)
	AssertNil(t, out)

	out, err = base64Sigil.Out(nil)
	RequireNoError(t, err)
	AssertNil(t, out)
}

func TestSigil_GzipSigil_Good(t *T) {
	gzipSigil := &GzipSigil{}
	data := []byte("the quick brown fox jumps over the lazy dog")

	compressed, err := gzipSigil.In(data)
	RequireNoError(t, err)
	AssertNotEqual(t, data, compressed)

	decompressed, err := gzipSigil.Out(compressed)
	RequireNoError(t, err)
	AssertEqual(t, data, decompressed)
}

func TestSigil_GzipSigil_Bad(t *T) {
	gzipSigil := &GzipSigil{}

	_, err := gzipSigil.Out([]byte("not gzip"))
	AssertError(t, err)

	compressed, err := gzipSigil.In([]byte{})
	RequireNoError(t, err)
	AssertNotEmpty(t, compressed)

	decompressed, err := gzipSigil.Out(compressed)
	RequireNoError(t, err)
	AssertEqual(t, []byte{}, decompressed)
}

func TestSigil_GzipSigil_NilInput_Good(t *T) {
	gzipSigil := &GzipSigil{}

	out, err := gzipSigil.In(nil)
	RequireNoError(t, err)
	AssertNil(t, out)

	out, err = gzipSigil.Out(nil)
	RequireNoError(t, err)
	AssertNil(t, out)
}

func TestSigil_JSONSigil_Good(t *T) {
	jsonSigil := &JSONSigil{Indent: false}
	data := []byte(`{  "key" :   "value"  }`)

	compacted, err := jsonSigil.In(data)
	RequireNoError(t, err)
	AssertEqual(t, []byte(`{"key":"value"}`), compacted)

	passthrough, err := jsonSigil.Out(compacted)
	RequireNoError(t, err)
	AssertEqual(t, compacted, passthrough)
}

func TestSigil_JSONSigil_Indent_Good(t *T) {
	jsonSigil := &JSONSigil{Indent: true}
	data := []byte(`{"key":"value"}`)

	indented, err := jsonSigil.In(data)
	RequireNoError(t, err)
	AssertContains(t, string(indented), "\n")
	AssertContains(t, string(indented), "  ")
}

func TestSigil_JSONSigil_Bad(t *T) {
	jsonSigil := &JSONSigil{Indent: false}

	_, err := jsonSigil.In([]byte("not json"))
	AssertError(t, err)
}

func TestSigil_JSONSigil_NilInput_Good(t *T) {
	jsonSigil := &JSONSigil{Indent: false}

	out, err := jsonSigil.In(nil)
	RequireNoError(t, err)
	AssertNil(t, out)

	out, err = jsonSigil.Out(nil)
	RequireNoError(t, err)
	AssertNil(t, out)
}

func TestSigil_HashSigil_Good(t *T) {
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
		t.Run(tt.name, func(t *T) {
			sigilValue, err := NewSigil(tt.sigilName)
			RequireNoError(t, err)

			hashed, err := sigilValue.In(data)
			RequireNoError(t, err)
			AssertLen(t, hashed, tt.size)

			passthrough, err := sigilValue.Out(hashed)
			RequireNoError(t, err)
			AssertEqual(t, hashed, passthrough)
		})
	}
}

func TestSigil_HashSigil_Bad(t *T) {
	hashSigil := &HashSigil{Hash: 0}
	_, err := hashSigil.In([]byte("data"))
	AssertError(t, err)
	AssertContains(t, err.Error(), "not available")
}

func TestSigil_HashSigil_EmptyInput_Good(t *T) {
	sigilValue, err := NewSigil("sha256")
	RequireNoError(t, err)

	hashed, err := sigilValue.In([]byte{})
	RequireNoError(t, err)
	AssertLen(t, hashed, sha256.Size)
}

func TestSigil_NewSigil_Good(t *T) {
	names := []string{
		"reverse", "hex", "base64", "gzip", "json", "json-indent",
		"md4", "md5", "sha1", "sha224", "sha256", "sha384", "sha512",
		"ripemd160",
		"sha3-224", "sha3-256", "sha3-384", "sha3-512",
		"sha512-224", "sha512-256",
		"blake2s-256", "blake2b-256", "blake2b-384", "blake2b-512",
	}

	for _, name := range names {
		t.Run(name, func(t *T) {
			sigilValue, err := NewSigil(name)
			RequireNoError(t, err)
			AssertNotNil(t, sigilValue)
		})
	}
}

func TestSigil_NewSigil_Bad(t *T) {
	_, err := NewSigil("nonexistent")
	AssertError(t, err)
	AssertContains(t, err.Error(), "unknown sigil name")
}

func TestSigil_NewSigil_KeylessScheme_Good(t *T) {
	sigilValue, err := NewSigil("hex")
	RequireNoError(t, err)
	AssertNotNil(t, sigilValue)
}

func TestSigil_NewSigil_ChaChaPoly1305RequiresKey_Bad(t *T) {
	_, err := NewSigil("chacha20poly1305")
	AssertError(t, err)
	if err == nil {
		t.Fatal("expected key material error")
	}
	AssertContains(t, err.Error(), "scheme requires key material; use NewChaChaPolySigil")
}

func TestSigil_NewSigil_EmptyName_Bad(t *T) {
	sigilValue, err := NewSigil("")
	AssertNil(t, sigilValue)
	AssertError(t, err)
	if err == nil {
		t.Fatal("expected empty sigil name to fail")
	}
	AssertContains(t, err.Error(), "unknown sigil name")
}

func TestSigil_Transmute_Good(t *T) {
	data := []byte("round trip")

	hexSigil, err := NewSigil("hex")
	RequireNoError(t, err)
	base64Sigil, err := NewSigil("base64")
	RequireNoError(t, err)

	chain := []Sigil{hexSigil, base64Sigil}

	encoded, err := Transmute(data, chain)
	RequireNoError(t, err)
	AssertNotEqual(t, data, encoded)

	decoded, err := Untransmute(encoded, chain)
	RequireNoError(t, err)
	AssertEqual(t, data, decoded)
}

func TestSigil_Transmute_MultiSigil_Good(t *T) {
	data := []byte("multi sigil pipeline test data")

	reverseSigil, err := NewSigil("reverse")
	RequireNoError(t, err)
	hexSigil, err := NewSigil("hex")
	RequireNoError(t, err)
	base64Sigil, err := NewSigil("base64")
	RequireNoError(t, err)

	chain := []Sigil{reverseSigil, hexSigil, base64Sigil}

	encoded, err := Transmute(data, chain)
	RequireNoError(t, err)

	decoded, err := Untransmute(encoded, chain)
	RequireNoError(t, err)
	AssertEqual(t, data, decoded)
}

func TestSigil_Transmute_GzipRoundTrip_Good(t *T) {
	data := []byte("compress then encode then decode then decompress")

	gzipSigil, err := NewSigil("gzip")
	RequireNoError(t, err)
	hexSigil, err := NewSigil("hex")
	RequireNoError(t, err)

	chain := []Sigil{gzipSigil, hexSigil}

	encoded, err := Transmute(data, chain)
	RequireNoError(t, err)

	decoded, err := Untransmute(encoded, chain)
	RequireNoError(t, err)
	AssertEqual(t, data, decoded)
}

func TestSigil_Transmute_Bad(t *T) {
	hexSigil := &HexSigil{}

	_, err := Untransmute([]byte("not-hex!!"), []Sigil{hexSigil})
	AssertError(t, err)
}

func TestSigil_Transmute_NilAndEmptyInput_Good(t *T) {
	data := []byte("unchanged")

	result, err := Transmute(data, nil)
	RequireNoError(t, err)
	AssertEqual(t, data, result)

	result, err = Untransmute(data, nil)
	RequireNoError(t, err)
	AssertEqual(t, data, result)

	hexSigil, _ := NewSigil("hex")
	result, err = Transmute(nil, []Sigil{hexSigil})
	RequireNoError(t, err)
	AssertNil(t, result)
}
