package sigil

import (
	"crypto"

	core "dappco.re/go"
)

var ax7Key = []byte("0123456789abcdef0123456789abcdef")

func TestAX7_Transmute_Good(t *core.T) {
	encoded, err := Transmute([]byte("payload"), []Sigil{&ReverseSigil{}, &HexSigil{}})
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("64616f6c796170"), encoded)
}

func TestAX7_Transmute_Bad(t *core.T) {
	_, err := Transmute([]byte("payload"), []Sigil{&HashSigil{Hash: 0}})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "sigil in failed")
}

func TestAX7_Transmute_Ugly(t *core.T) {
	encoded, err := Transmute(nil, []Sigil{&ReverseSigil{}, &HexSigil{}})
	core.AssertNoError(t, err)
	core.AssertNil(t, encoded)
}

func TestAX7_Untransmute_Good(t *core.T) {
	decoded, err := Untransmute([]byte("64616f6c796170"), []Sigil{&ReverseSigil{}, &HexSigil{}})
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("payload"), decoded)
}

func TestAX7_Untransmute_Bad(t *core.T) {
	_, err := Untransmute([]byte("not hex"), []Sigil{&HexSigil{}})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "sigil out failed")
}

func TestAX7_Untransmute_Ugly(t *core.T) {
	decoded, err := Untransmute(nil, []Sigil{&ReverseSigil{}, &HexSigil{}})
	core.AssertNoError(t, err)
	core.AssertNil(t, decoded)
}

func TestAX7_ReverseSigil_In_Good(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.In([]byte("abc"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("cba"), got)
}

func TestAX7_ReverseSigil_In_Bad(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.In(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_ReverseSigil_In_Ugly(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.In([]byte(""))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte{}, got)
}

func TestAX7_ReverseSigil_Out_Good(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.Out([]byte("cba"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("abc"), got)
}

func TestAX7_ReverseSigil_Out_Bad(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_ReverseSigil_Out_Ugly(t *core.T) {
	sigilValue := &ReverseSigil{}
	got, err := sigilValue.Out([]byte(""))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte{}, got)
}

func TestAX7_HexSigil_In_Good(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.In([]byte("hi"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("6869"), got)
}

func TestAX7_HexSigil_In_Bad(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.In(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_HexSigil_In_Ugly(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.In([]byte{})
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte{}, got)
}

func TestAX7_HexSigil_Out_Good(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.Out([]byte("6869"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("hi"), got)
}

func TestAX7_HexSigil_Out_Bad(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.Out([]byte("zz"))
	core.AssertError(t, err)
	core.AssertEqual(t, []byte{0}, got)
}

func TestAX7_HexSigil_Out_Ugly(t *core.T) {
	sigilValue := &HexSigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_Base64Sigil_In_Good(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.In([]byte("hi"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("aGk="), got)
}

func TestAX7_Base64Sigil_In_Bad(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.In(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_Base64Sigil_In_Ugly(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.In([]byte{})
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte{}, got)
}

func TestAX7_Base64Sigil_Out_Good(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.Out([]byte("aGk="))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("hi"), got)
}

func TestAX7_Base64Sigil_Out_Bad(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.Out([]byte("!!!"))
	core.AssertError(t, err)
	core.AssertEmpty(t, got)
}

func TestAX7_Base64Sigil_Out_Ugly(t *core.T) {
	sigilValue := &Base64Sigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_GzipSigil_In_Good(t *core.T) {
	sigilValue := &GzipSigil{}
	got, err := sigilValue.In([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, got)
}

func TestAX7_GzipSigil_In_Bad(t *core.T) {
	sigilValue := &GzipSigil{}
	got, err := sigilValue.In(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_GzipSigil_In_Ugly(t *core.T) {
	buffer := &sigilBuffer{}
	sigilValue := &GzipSigil{outputWriter: buffer}
	got, err := sigilValue.In([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
	core.AssertNotEmpty(t, buffer.Bytes())
}

func TestAX7_GzipSigil_Out_Good(t *core.T) {
	sigilValue := &GzipSigil{}
	compressed, err := sigilValue.In([]byte("payload"))
	core.RequireNoError(t, err)
	got, err := sigilValue.Out(compressed)
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("payload"), got)
}

func TestAX7_GzipSigil_Out_Bad(t *core.T) {
	sigilValue := &GzipSigil{}
	got, err := sigilValue.Out([]byte("not gzip"))
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_GzipSigil_Out_Ugly(t *core.T) {
	sigilValue := &GzipSigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_Buffer_Write_Good(t *core.T) {
	buffer := &sigilBuffer{}
	count, err := buffer.Write([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, len("payload"), count)
}

func TestAX7_Buffer_Write_Bad(t *core.T) {
	buffer := &sigilBuffer{}
	count, err := buffer.Write(nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_Buffer_Write_Ugly(t *core.T) {
	buffer := &sigilBuffer{data: []byte("a")}
	count, err := buffer.Write([]byte("b"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}

func TestAX7_Buffer_Bytes_Good(t *core.T) {
	buffer := &sigilBuffer{data: []byte("payload")}
	got := buffer.Bytes()
	core.AssertEqual(t, []byte("payload"), got)
}

func TestAX7_Buffer_Bytes_Bad(t *core.T) {
	buffer := &sigilBuffer{}
	got := buffer.Bytes()
	core.AssertNil(t, got)
}

func TestAX7_Buffer_Bytes_Ugly(t *core.T) {
	buffer := &sigilBuffer{data: []byte{}}
	got := buffer.Bytes()
	core.AssertEqual(t, []byte{}, got)
}

func TestAX7_JSONSigil_In_Good(t *core.T) {
	sigilValue := &JSONSigil{}
	got, err := sigilValue.In([]byte(`{ "key" : "value" }`))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte(`{"key":"value"}`), got)
}

func TestAX7_JSONSigil_In_Bad(t *core.T) {
	sigilValue := &JSONSigil{}
	got, err := sigilValue.In([]byte("not json"))
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_JSONSigil_In_Ugly(t *core.T) {
	sigilValue := &JSONSigil{Indent: true}
	got, err := sigilValue.In([]byte(`{"key":"value"}`))
	core.AssertNoError(t, err)
	core.AssertContains(t, string(got), "\n")
}

func TestAX7_JSONSigil_Out_Good(t *core.T) {
	sigilValue := &JSONSigil{}
	got, err := sigilValue.Out([]byte(`{"key":"value"}`))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte(`{"key":"value"}`), got)
}

func TestAX7_JSONSigil_Out_Bad(t *core.T) {
	sigilValue := &JSONSigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_JSONSigil_Out_Ugly(t *core.T) {
	sigilValue := &JSONSigil{Indent: true}
	got, err := sigilValue.Out([]byte("not json"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("not json"), got)
}

func TestAX7_NewHashSigil_Good(t *core.T) {
	sigilValue := NewHashSigil(crypto.SHA256)
	core.AssertNotNil(t, sigilValue)
	core.AssertEqual(t, crypto.SHA256, sigilValue.Hash)
}

func TestAX7_NewHashSigil_Bad(t *core.T) {
	sigilValue := NewHashSigil(crypto.Hash(0))
	_, err := sigilValue.In([]byte("payload"))
	core.AssertError(t, err)
}

func TestAX7_NewHashSigil_Ugly(t *core.T) {
	sigilValue := NewHashSigil(crypto.MD5)
	got, err := sigilValue.In([]byte{})
	core.AssertNoError(t, err)
	core.AssertLen(t, got, 16)
}

func TestAX7_HashSigil_In_Good(t *core.T) {
	sigilValue := &HashSigil{Hash: crypto.SHA256}
	got, err := sigilValue.In([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertLen(t, got, 32)
}

func TestAX7_HashSigil_In_Bad(t *core.T) {
	sigilValue := &HashSigil{Hash: crypto.Hash(0)}
	got, err := sigilValue.In([]byte("payload"))
	core.AssertError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_HashSigil_In_Ugly(t *core.T) {
	sigilValue := &HashSigil{Hash: crypto.SHA512}
	got, err := sigilValue.In(nil)
	core.AssertNoError(t, err)
	core.AssertLen(t, got, 64)
}

func TestAX7_HashSigil_Out_Good(t *core.T) {
	sigilValue := &HashSigil{Hash: crypto.SHA256}
	got, err := sigilValue.Out([]byte("digest"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("digest"), got)
}

func TestAX7_HashSigil_Out_Bad(t *core.T) {
	sigilValue := &HashSigil{}
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_HashSigil_Out_Ugly(t *core.T) {
	sigilValue := &HashSigil{Hash: crypto.MD5}
	got, err := sigilValue.Out([]byte{})
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte{}, got)
}

func TestAX7_NewSigil_Good(t *core.T) {
	sigilValue, err := NewSigil("hex")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, sigilValue)
}

func TestAX7_NewSigil_Bad(t *core.T) {
	sigilValue, err := NewSigil("missing")
	core.AssertError(t, err)
	core.AssertNil(t, sigilValue)
}

func TestAX7_NewSigil_Ugly(t *core.T) {
	sigilValue, err := NewSigil("chacha20poly1305")
	core.AssertError(t, err)
	core.AssertNil(t, sigilValue)
}

func TestAX7_XORObfuscator_Obfuscate_Good(t *core.T) {
	obfuscator := &XORObfuscator{}
	got := obfuscator.Obfuscate([]byte("payload"), []byte("nonce"))
	core.AssertNotEqual(t, []byte("payload"), got)
	core.AssertLen(t, got, len("payload"))
}

func TestAX7_XORObfuscator_Obfuscate_Bad(t *core.T) {
	obfuscator := &XORObfuscator{}
	got := obfuscator.Obfuscate(nil, []byte("nonce"))
	core.AssertNil(t, got)
}

func TestAX7_XORObfuscator_Obfuscate_Ugly(t *core.T) {
	obfuscator := &XORObfuscator{}
	got := obfuscator.Obfuscate([]byte{}, []byte("nonce"))
	core.AssertEqual(t, []byte{}, got)
}

func TestAX7_XORObfuscator_Deobfuscate_Good(t *core.T) {
	obfuscator := &XORObfuscator{}
	encoded := obfuscator.Obfuscate([]byte("payload"), []byte("nonce"))
	got := obfuscator.Deobfuscate(encoded, []byte("nonce"))
	core.AssertEqual(t, []byte("payload"), got)
}

func TestAX7_XORObfuscator_Deobfuscate_Bad(t *core.T) {
	obfuscator := &XORObfuscator{}
	got := obfuscator.Deobfuscate(nil, []byte("nonce"))
	core.AssertNil(t, got)
}

func TestAX7_XORObfuscator_Deobfuscate_Ugly(t *core.T) {
	obfuscator := &XORObfuscator{}
	got := obfuscator.Deobfuscate([]byte{}, []byte("nonce"))
	core.AssertEqual(t, []byte{}, got)
}

func TestAX7_ShuffleMaskObfuscator_Obfuscate_Good(t *core.T) {
	obfuscator := &ShuffleMaskObfuscator{}
	got := obfuscator.Obfuscate([]byte("payload"), []byte("nonce"))
	core.AssertNotEqual(t, []byte("payload"), got)
	core.AssertLen(t, got, len("payload"))
}

func TestAX7_ShuffleMaskObfuscator_Obfuscate_Bad(t *core.T) {
	obfuscator := &ShuffleMaskObfuscator{}
	got := obfuscator.Obfuscate(nil, []byte("nonce"))
	core.AssertNil(t, got)
}

func TestAX7_ShuffleMaskObfuscator_Obfuscate_Ugly(t *core.T) {
	obfuscator := &ShuffleMaskObfuscator{}
	got := obfuscator.Obfuscate([]byte{1}, []byte("nonce"))
	core.AssertLen(t, got, 1)
}

func TestAX7_ShuffleMaskObfuscator_Deobfuscate_Good(t *core.T) {
	obfuscator := &ShuffleMaskObfuscator{}
	encoded := obfuscator.Obfuscate([]byte("payload"), []byte("nonce"))
	got := obfuscator.Deobfuscate(encoded, []byte("nonce"))
	core.AssertEqual(t, []byte("payload"), got)
}

func TestAX7_ShuffleMaskObfuscator_Deobfuscate_Bad(t *core.T) {
	obfuscator := &ShuffleMaskObfuscator{}
	got := obfuscator.Deobfuscate(nil, []byte("nonce"))
	core.AssertNil(t, got)
}

func TestAX7_ShuffleMaskObfuscator_Deobfuscate_Ugly(t *core.T) {
	obfuscator := &ShuffleMaskObfuscator{}
	encoded := obfuscator.Obfuscate([]byte{1}, []byte("nonce"))
	got := obfuscator.Deobfuscate(encoded, []byte("nonce"))
	core.AssertEqual(t, []byte{1}, got)
}

func TestAX7_NewChaChaPolySigil_Good(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, sigilValue)
}

func TestAX7_NewChaChaPolySigil_Bad(t *core.T) {
	sigilValue, err := NewChaChaPolySigil([]byte("short"), nil)
	core.AssertErrorIs(t, err, InvalidKeyError)
	core.AssertNil(t, sigilValue)
}

func TestAX7_NewChaChaPolySigil_Ugly(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, &ShuffleMaskObfuscator{})
	core.AssertNoError(t, err)
	core.AssertNotNil(t, sigilValue.Obfuscator())
}

func TestAX7_ChaChaPolySigil_Key_Good(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.RequireNoError(t, err)
	got := sigilValue.Key()
	core.AssertEqual(t, ax7Key, got)
}

func TestAX7_ChaChaPolySigil_Key_Bad(t *core.T) {
	sigilValue := &ChaChaPolySigil{}
	got := sigilValue.Key()
	core.AssertEmpty(t, got)
}

func TestAX7_ChaChaPolySigil_Key_Ugly(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.RequireNoError(t, err)
	got := sigilValue.Key()
	got[0] ^= 0xff
	core.AssertEqual(t, ax7Key, sigilValue.Key())
}

func TestAX7_ChaChaPolySigil_Nonce_Good(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.RequireNoError(t, err)
	got := sigilValue.Nonce()
	core.AssertNil(t, got)
}

func TestAX7_ChaChaPolySigil_Nonce_Bad(t *core.T) {
	sigilValue := &ChaChaPolySigil{}
	got := sigilValue.Nonce()
	core.AssertNil(t, got)
}

func TestAX7_ChaChaPolySigil_Nonce_Ugly(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, &ShuffleMaskObfuscator{})
	core.RequireNoError(t, err)
	got := sigilValue.Nonce()
	core.AssertNil(t, got)
}

func TestAX7_ChaChaPolySigil_Obfuscator_Good(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.RequireNoError(t, err)
	got := sigilValue.Obfuscator()
	core.AssertNotNil(t, got)
}

func TestAX7_ChaChaPolySigil_Obfuscator_Bad(t *core.T) {
	sigilValue := &ChaChaPolySigil{}
	got := sigilValue.Obfuscator()
	core.AssertNil(t, got)
}

func TestAX7_ChaChaPolySigil_Obfuscator_Ugly(t *core.T) {
	obfuscator := &ShuffleMaskObfuscator{}
	sigilValue, err := NewChaChaPolySigil(ax7Key, obfuscator)
	core.RequireNoError(t, err)
	core.AssertSame(t, obfuscator, sigilValue.Obfuscator())
}

func TestAX7_ChaChaPolySigil_SetObfuscator_Good(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.RequireNoError(t, err)
	obfuscator := &ShuffleMaskObfuscator{}
	sigilValue.SetObfuscator(obfuscator)
	core.AssertSame(t, obfuscator, sigilValue.Obfuscator())
}

func TestAX7_ChaChaPolySigil_SetObfuscator_Bad(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.RequireNoError(t, err)
	sigilValue.SetObfuscator(nil)
	core.AssertNil(t, sigilValue.Obfuscator())
}

func TestAX7_ChaChaPolySigil_SetObfuscator_Ugly(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, &ShuffleMaskObfuscator{})
	core.RequireNoError(t, err)
	sigilValue.SetObfuscator(&XORObfuscator{})
	core.AssertNotNil(t, sigilValue.Obfuscator())
}

func TestAX7_ChaChaPolySigil_In_Good(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.RequireNoError(t, err)
	got, err := sigilValue.In([]byte("payload"))
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, got)
}

func TestAX7_ChaChaPolySigil_In_Bad(t *core.T) {
	sigilValue := &ChaChaPolySigil{}
	got, err := sigilValue.In([]byte("payload"))
	core.AssertErrorIs(t, err, NoKeyConfiguredError)
	core.AssertNil(t, got)
}

func TestAX7_ChaChaPolySigil_In_Ugly(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.RequireNoError(t, err)
	got, err := sigilValue.In(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_ChaChaPolySigil_Out_Good(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.RequireNoError(t, err)
	ciphertext, err := sigilValue.In([]byte("payload"))
	core.RequireNoError(t, err)
	got, err := sigilValue.Out(ciphertext)
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("payload"), got)
}

func TestAX7_ChaChaPolySigil_Out_Bad(t *core.T) {
	sigilValue := &ChaChaPolySigil{}
	got, err := sigilValue.Out([]byte("ciphertext"))
	core.AssertErrorIs(t, err, NoKeyConfiguredError)
	core.AssertNil(t, got)
}

func TestAX7_ChaChaPolySigil_Out_Ugly(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.RequireNoError(t, err)
	got, err := sigilValue.Out(nil)
	core.AssertNoError(t, err)
	core.AssertNil(t, got)
}

func TestAX7_NonceFromCiphertext_Good(t *core.T) {
	sigilValue, err := NewChaChaPolySigil(ax7Key, nil)
	core.RequireNoError(t, err)
	ciphertext, err := sigilValue.In([]byte("payload"))
	core.RequireNoError(t, err)
	nonce, err := NonceFromCiphertext(ciphertext)
	core.AssertNoError(t, err)
	core.AssertLen(t, nonce, 24)
}

func TestAX7_NonceFromCiphertext_Bad(t *core.T) {
	nonce, err := NonceFromCiphertext([]byte("short"))
	core.AssertErrorIs(t, err, CiphertextTooShortError)
	core.AssertNil(t, nonce)
}

func TestAX7_NonceFromCiphertext_Ugly(t *core.T) {
	nonce, err := NonceFromCiphertext(make([]byte, 24))
	core.AssertNoError(t, err)
	core.AssertLen(t, nonce, 24)
}
